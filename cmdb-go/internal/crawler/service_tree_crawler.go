package crawler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"cmdb-crawler/internal/client"
	"cmdb-crawler/internal/models"

	"go.uber.org/zap"
)

// ServiceTreeCrawler 服务树爬取器
type ServiceTreeCrawler struct {
	client          *client.CMDBClient
	logger          *zap.Logger
	maxDepth        int
	pageSize        int
	maxWorkers      int
	includeStats    bool
	requestInterval time.Duration
}

// NewServiceTreeCrawler 创建服务树爬取器
func NewServiceTreeCrawler(client *client.CMDBClient, logger *zap.Logger) *ServiceTreeCrawler {
	return &ServiceTreeCrawler{
		client:          client,
		logger:          logger,
		maxDepth:        -1, // 无限制
		pageSize:        1000,
		maxWorkers:      10,
		includeStats:    true,
		requestInterval: 100 * time.Millisecond,
	}
}

// SetMaxDepth 设置最大爬取深度
func (c *ServiceTreeCrawler) SetMaxDepth(depth int) *ServiceTreeCrawler {
	c.maxDepth = depth
	return c
}

// SetPageSize 设置分页大小
func (c *ServiceTreeCrawler) SetPageSize(size int) *ServiceTreeCrawler {
	c.pageSize = size
	return c
}

// SetMaxWorkers 设置最大并发数
func (c *ServiceTreeCrawler) SetMaxWorkers(workers int) *ServiceTreeCrawler {
	c.maxWorkers = workers
	return c
}

// SetIncludeStats 设置是否包含统计信息
func (c *ServiceTreeCrawler) SetIncludeStats(include bool) *ServiceTreeCrawler {
	c.includeStats = include
	return c
}

// SetRequestInterval 设置请求间隔
func (c *ServiceTreeCrawler) SetRequestInterval(interval time.Duration) *ServiceTreeCrawler {
	c.requestInterval = interval
	return c
}

// CrawlAllServiceTrees 爬取所有服务树
func (c *ServiceTreeCrawler) CrawlAllServiceTrees(ctx context.Context) ([]*models.ServiceTreeData, error) {
	c.logger.Info("Starting to crawl all service trees")

	// 获取服务树视图列表
	viewsResp, err := c.client.GetRelationViews()
	if err != nil {
		return nil, fmt.Errorf("failed to get relation views: %w", err)
	}

	if len(viewsResp.Views) == 0 {
		c.logger.Warn("No service tree views found")
		return []*models.ServiceTreeData{}, nil
	}

	var results []*models.ServiceTreeData
	for viewName, viewConfig := range viewsResp.Views {
		// 从name2id中获取view ID
		viewID := c.findViewIDByName(viewName, viewsResp.Name2ID)

		c.logger.Info("Crawling service tree",
			zap.String("view_name", viewName),
			zap.Int("view_id", viewID))

		treeData, err := c.CrawlServiceTree(ctx, viewName, viewID, viewConfig, viewsResp.ID2Type)
		if err != nil {
			c.logger.Error("Failed to crawl service tree",
				zap.String("view_name", viewName),
				zap.Error(err))
			continue
		}

		results = append(results, treeData)
	}

	c.logger.Info("Completed crawling all service trees",
		zap.Int("total_trees", len(results)))

	return results, nil
}

// CrawlServiceTree 爬取指定的服务树
func (c *ServiceTreeCrawler) CrawlServiceTree(ctx context.Context, viewName string, viewID int,
	viewConfig models.ServiceTreeView, id2Type map[string]models.CIType) (*models.ServiceTreeData, error) {

	c.logger.Info("Starting to crawl service tree",
		zap.String("view_name", viewName),
		zap.Int("levels", len(viewConfig.Topo)))

	if len(viewConfig.Topo) == 0 {
		return nil, fmt.Errorf("service tree has no levels defined")
	}

	// 创建服务树数据结构
	treeData := &models.ServiceTreeData{
		ViewName:  viewName,
		ViewID:    viewID,
		Config:    viewConfig,
		RootNodes: make([]*models.ServiceTreeNode, 0),
		CrawledAt: time.Now(),
	}

	// 获取根节点类型
	rootTypeIDs := viewConfig.Topo[0]
	c.logger.Info("Loading root nodes",
		zap.Ints("root_type_ids", rootTypeIDs))

	// 查询根节点实例
	query := c.client.BuildCITypeQuery(rootTypeIDs)
	rootResp, err := c.client.SearchCI(query, c.pageSize, false)
	if err != nil {
		return nil, fmt.Errorf("failed to search root nodes: %w", err)
	}

	if len(rootResp.Result) == 0 {
		c.logger.Warn("No root nodes found for service tree",
			zap.String("view_name", viewName))
		return treeData, nil
	}

	// 构建根节点
	rootNodes := make([]*models.ServiceTreeNode, 0, len(rootResp.Result))
	for _, ci := range rootResp.Result {
		ciType, exists := id2Type[strconv.Itoa(ci.Type)]
		typeName := ""
		if exists {
			typeName = ciType.Alias
			if typeName == "" {
				typeName = ciType.Name
			}
		}

		node := &models.ServiceTreeNode{
			ID:         ci.ID,
			Type:       ci.Type,
			TypeName:   typeName,
			Name:       ci.GetDisplayName(),
			Level:      0,
			Children:   make([]*models.ServiceTreeNode, 0),
			IsLeaf:     false,
			Attributes: ci.Attrs,
		}

		rootNodes = append(rootNodes, node)
	}

	// 如果需要统计信息，获取根节点的子节点统计
	if c.includeStats && len(viewConfig.Leaf) > 0 {
		err = c.loadRootNodeStatistics(rootNodes, viewConfig)
		if err != nil {
			c.logger.Warn("Failed to load root node statistics", zap.Error(err))
		}
	}

	// 并发爬取每个根节点的子树
	var wg sync.WaitGroup
	errChan := make(chan error, len(rootNodes))
	semaphore := make(chan struct{}, c.maxWorkers)

	for _, rootNode := range rootNodes {
		wg.Add(1)
		go func(node *models.ServiceTreeNode) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := c.crawlNodeChildren(ctx, node, viewConfig, id2Type, 1); err != nil {
				errChan <- fmt.Errorf("failed to crawl children for node %s: %w", node.Name, err)
			}
		}(rootNode)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var crawlErrors []error
	for err := range errChan {
		crawlErrors = append(crawlErrors, err)
	}

	if len(crawlErrors) > 0 {
		c.logger.Warn("Some nodes failed to crawl",
			zap.Int("error_count", len(crawlErrors)))
		for _, err := range crawlErrors {
			c.logger.Error("Crawl error", zap.Error(err))
		}
	}

	treeData.RootNodes = rootNodes
	treeData.CountNodes()
	treeData.CalculateMaxDepth()

	c.logger.Info("Successfully crawled service tree",
		zap.String("view_name", viewName),
		zap.Int("total_nodes", treeData.TotalNodes),
		zap.Int("max_depth", treeData.MaxDepth))

	return treeData, nil
}

// crawlNodeChildren 递归爬取节点的子节点
func (c *ServiceTreeCrawler) crawlNodeChildren(ctx context.Context, node *models.ServiceTreeNode,
	viewConfig models.ServiceTreeView, id2Type map[string]models.CIType, currentLevel int) error {

	// 检查是否达到最大深度
	if c.maxDepth > 0 && currentLevel >= c.maxDepth {
		return nil
	}

	// 检查是否超出视图定义的层级
	if currentLevel >= len(viewConfig.Topo) {
		node.IsLeaf = true
		return nil
	}

	// 获取当前级别的子类型
	childTypeIDs := viewConfig.Topo[currentLevel]
	if len(childTypeIDs) == 0 {
		node.IsLeaf = true
		return nil
	}

	// 请求间隔控制
	time.Sleep(c.requestInterval)

	// 构建查询参数
	params := map[string]interface{}{
		"q":       c.client.BuildCITypeQuery(childTypeIDs),
		"root_id": node.ID,
		"level":   1,
		"count":   c.pageSize,
	}

	// 添加descendant_ids参数
	if len(viewConfig.TopoFlatten) > currentLevel+1 {
		descendantIDs := viewConfig.TopoFlatten[currentLevel+1:]
		descendantStrs := make([]string, len(descendantIDs))
		for i, id := range descendantIDs {
			descendantStrs[i] = strconv.Itoa(id)
		}
		params["descendant_ids"] = strings.Join(descendantStrs, ",")
	}

	// 搜索子节点
	childResp, err := c.client.SearchCIRelation(params)
	if err != nil {
		return fmt.Errorf("failed to search children for node %d: %w", node.ID, err)
	}

	if len(childResp.Result) == 0 {
		node.IsLeaf = true
		return nil
	}

	// 构建子节点
	for _, ci := range childResp.Result {
		ciType, exists := id2Type[strconv.Itoa(ci.Type)]
		typeName := ""
		if exists {
			typeName = ciType.Alias
			if typeName == "" {
				typeName = ciType.Name
			}
		}

		child := &models.ServiceTreeNode{
			ID:         ci.ID,
			Type:       ci.Type,
			TypeName:   typeName,
			Name:       ci.GetDisplayName(),
			Level:      currentLevel,
			Children:   make([]*models.ServiceTreeNode, 0),
			IsLeaf:     false,
			Attributes: ci.Attrs,
		}

		node.AddChild(child)

		// 递归爬取子节点的子节点
		if err := c.crawlNodeChildren(ctx, child, viewConfig, id2Type, currentLevel+1); err != nil {
			c.logger.Error("Failed to crawl grandchildren",
				zap.Int("parent_id", node.ID),
				zap.Int("child_id", child.ID),
				zap.Error(err))
		}
	}

	node.ChildCount = len(node.Children)
	return nil
}

// loadRootNodeStatistics 加载根节点统计信息
func (c *ServiceTreeCrawler) loadRootNodeStatistics(rootNodes []*models.ServiceTreeNode,
	viewConfig models.ServiceTreeView) error {

	if len(rootNodes) == 0 || len(viewConfig.Leaf) == 0 {
		return nil
	}

	// 构建root_ids参数
	rootIDs := make([]string, len(rootNodes))
	for i, node := range rootNodes {
		rootIDs[i] = strconv.Itoa(node.ID)
	}

	// 计算到叶子节点的层级深度
	level := len(viewConfig.TopoFlatten)

	// 构建统计查询参数
	params := map[string]interface{}{
		"root_ids": strings.Join(rootIDs, ","),
		"level":    level,
		"type_ids": viewConfig.Leaf,
		"has_m2m":  0,
	}

	// 获取统计信息
	stats, err := c.client.GetCIRelationStatistics(params)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	// 将统计信息设置到对应的根节点
	for _, node := range rootNodes {
		count := stats.GetCount(strconv.Itoa(node.ID))
		node.Statistics = map[string]int{
			"total_descendants": count,
		}
	}

	return nil
}

// findViewIDByName 根据视图名称查找视图ID
func (c *ServiceTreeCrawler) findViewIDByName(viewName string, name2id [][]interface{}) int {
	for _, pair := range name2id {
		if len(pair) >= 2 {
			if name, ok := pair[0].(string); ok && name == viewName {
				if id, ok := pair[1].(float64); ok {
					return int(id)
				}
			}
		}
	}
	return -1
}

// CrawlSpecificViews 爬取指定的服务树视图
func (c *ServiceTreeCrawler) CrawlSpecificViews(ctx context.Context, targetViews []string) ([]*models.ServiceTreeData, error) {
	if len(targetViews) == 0 {
		return c.CrawlAllServiceTrees(ctx)
	}

	c.logger.Info("Crawling specific service trees", zap.Strings("target_views", targetViews))

	// 获取服务树视图列表
	viewsResp, err := c.client.GetRelationViews()
	if err != nil {
		return nil, fmt.Errorf("failed to get relation views: %w", err)
	}

	var results []*models.ServiceTreeData
	targetSet := make(map[string]bool)
	for _, view := range targetViews {
		targetSet[view] = true
	}

	for viewName, viewConfig := range viewsResp.Views {
		if !targetSet[viewName] {
			continue
		}

		viewID := c.findViewIDByName(viewName, viewsResp.Name2ID)

		c.logger.Info("Crawling specific service tree",
			zap.String("view_name", viewName),
			zap.Int("view_id", viewID))

		treeData, err := c.CrawlServiceTree(ctx, viewName, viewID, viewConfig, viewsResp.ID2Type)
		if err != nil {
			c.logger.Error("Failed to crawl specific service tree",
				zap.String("view_name", viewName),
				zap.Error(err))
			continue
		}

		results = append(results, treeData)
	}

	// 检查是否有未找到的视图
	for _, target := range targetViews {
		found := false
		for _, result := range results {
			if result.ViewName == target {
				found = true
				break
			}
		}
		if !found {
			c.logger.Warn("Target service tree view not found", zap.String("view_name", target))
		}
	}

	c.logger.Info("Completed crawling specific service trees",
		zap.Int("requested", len(targetViews)),
		zap.Int("found", len(results)))

	return results, nil
}
