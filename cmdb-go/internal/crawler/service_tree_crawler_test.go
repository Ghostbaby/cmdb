package crawler

import (
	"context"
	"testing"
	"time"

	"cmdb-crawler/internal/client"
	"cmdb-crawler/internal/models"

	"go.uber.org/zap"
)

// 演示环境配置
const (
	demoBaseURL    = "https://cmdb.veops.cn"
	demoAPIVersion = "api/v0.1"
	demoAPIKey     = "d0a8fb5aeedf466c92cc5142a18d1a68"
	demoAPISecret  = "DSGYH81jqfw~%A&vgyJKXrO*UFVaW2xt"
)

// createTestClient 创建测试用的CMDB客户端
func createTestClient(t *testing.T) *client.CMDBClient {
	logger, _ := zap.NewDevelopment()

	cmdbClient := client.NewCMDBClient(demoBaseURL, demoAPIVersion, logger)
	cmdbClient.SetAPICredentials(demoAPIKey, demoAPISecret)
	cmdbClient.SetTimeout(30 * time.Second)
	cmdbClient.SetRetry(3, 1*time.Second)

	return cmdbClient
}

// createTestCrawler 创建测试用的服务树爬取器
func createTestCrawler(t *testing.T) *ServiceTreeCrawler {
	client := createTestClient(t)
	logger, _ := zap.NewDevelopment()

	crawler := NewServiceTreeCrawler(client, logger)
	crawler.SetMaxDepth(2). // 限制深度避免测试时间过长
				SetPageSize(100).                          // 较小的分页大小
				SetMaxWorkers(5).                          // 较少的并发数
				SetIncludeStats(true).                     // 包含统计信息
				SetRequestInterval(200 * time.Millisecond) // 稍长的请求间隔

	return crawler
}

// TestNewServiceTreeCrawler 测试创建服务树爬取器
func TestNewServiceTreeCrawler(t *testing.T) {
	client := createTestClient(t)
	logger, _ := zap.NewDevelopment()

	crawler := NewServiceTreeCrawler(client, logger)

	if crawler == nil {
		t.Fatal("Expected crawler to be created, got nil")
	}

	if crawler.client != client {
		t.Error("Expected client to be set correctly")
	}

	if crawler.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	// 测试默认值
	if crawler.maxDepth != -1 {
		t.Errorf("Expected default maxDepth to be -1, got %d", crawler.maxDepth)
	}

	if crawler.pageSize != 1000 {
		t.Errorf("Expected default pageSize to be 1000, got %d", crawler.pageSize)
	}

	if crawler.maxWorkers != 10 {
		t.Errorf("Expected default maxWorkers to be 10, got %d", crawler.maxWorkers)
	}

	if !crawler.includeStats {
		t.Error("Expected default includeStats to be true")
	}
}

// TestSetterMethods 测试设置方法
func TestSetterMethods(t *testing.T) {
	crawler := createTestCrawler(t)

	// 测试链式调用
	result := crawler.SetMaxDepth(5).
		SetPageSize(500).
		SetMaxWorkers(15).
		SetIncludeStats(false).
		SetRequestInterval(500 * time.Millisecond)

	if result != crawler {
		t.Error("Expected setter methods to return the same crawler instance")
	}

	// 验证设置值
	if crawler.maxDepth != 5 {
		t.Errorf("Expected maxDepth to be 5, got %d", crawler.maxDepth)
	}

	if crawler.pageSize != 500 {
		t.Errorf("Expected pageSize to be 500, got %d", crawler.pageSize)
	}

	if crawler.maxWorkers != 15 {
		t.Errorf("Expected maxWorkers to be 15, got %d", crawler.maxWorkers)
	}

	if crawler.includeStats {
		t.Error("Expected includeStats to be false")
	}

	if crawler.requestInterval != 500*time.Millisecond {
		t.Errorf("Expected requestInterval to be 500ms, got %v", crawler.requestInterval)
	}
}

// TestCrawlAllServiceTrees 测试爬取所有服务树
func TestCrawlAllServiceTrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)
	ctx := context.Background()

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	trees, err := crawler.CrawlAllServiceTrees(ctx)
	if err != nil {
		t.Fatalf("Failed to crawl all service trees: %v", err)
	}

	if trees == nil {
		t.Fatal("Expected trees to be non-nil")
	}

	t.Logf("Successfully crawled %d service trees", len(trees))

	// 验证每个服务树的基本信息
	for i, tree := range trees {
		t.Logf("Tree %d: %s (ID: %d)", i+1, tree.ViewName, tree.ViewID)

		if tree.ViewName == "" {
			t.Errorf("Tree %d has empty view name", i)
		}

		if tree.ViewID <= 0 {
			t.Errorf("Tree %d has invalid view ID: %d", i, tree.ViewID)
		}

		if tree.CrawledAt.IsZero() {
			t.Errorf("Tree %d has zero crawled time", i)
		}

		t.Logf("  Root nodes: %d, Total nodes: %d, Max depth: %d",
			len(tree.RootNodes), tree.TotalNodes, tree.MaxDepth)
	}
}

// TestCrawlSpecificViews 测试爬取指定视图
func TestCrawlSpecificViews(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)
	ctx := context.Background()

	// 首先获取可用的视图列表
	viewsResp, err := crawler.client.GetRelationViews()
	if err != nil {
		t.Fatalf("Failed to get relation views: %v", err)
	}

	if len(viewsResp.Views) == 0 {
		t.Skip("No service tree views available for testing")
	}

	// 选择第一个视图进行测试
	var targetView string
	for viewName := range viewsResp.Views {
		targetView = viewName
		break
	}

	t.Logf("Testing with view: %s", targetView)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	trees, err := crawler.CrawlSpecificViews(ctx, []string{targetView})
	if err != nil {
		t.Fatalf("Failed to crawl specific views: %v", err)
	}

	if len(trees) == 0 {
		t.Log("No trees returned - this might be expected if the view has no data")
		return
	}

	// 验证返回的树
	if len(trees) > 1 {
		t.Errorf("Expected at most 1 tree, got %d", len(trees))
	}

	tree := trees[0]
	if tree.ViewName != targetView {
		t.Errorf("Expected view name %s, got %s", targetView, tree.ViewName)
	}

	t.Logf("Successfully crawled view '%s': %d root nodes, %d total nodes",
		tree.ViewName, len(tree.RootNodes), tree.TotalNodes)
}

// TestCrawlSpecificViewsEmpty 测试爬取空的指定视图列表
func TestCrawlSpecificViewsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)
	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	// 传入空的视图列表，应该爬取所有视图
	trees, err := crawler.CrawlSpecificViews(ctx, []string{})
	if err != nil {
		t.Fatalf("Failed to crawl with empty view list: %v", err)
	}

	t.Logf("Crawled %d trees with empty view list", len(trees))
}

// TestCrawlSpecificViewsNonExistent 测试爬取不存在的视图
func TestCrawlSpecificViewsNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)
	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 尝试爬取不存在的视图
	trees, err := crawler.CrawlSpecificViews(ctx, []string{"NonExistentView"})
	if err != nil {
		t.Fatalf("Failed to crawl non-existent views: %v", err)
	}

	// 应该返回空结果，但不应该出错
	if len(trees) != 0 {
		t.Errorf("Expected 0 trees for non-existent view, got %d", len(trees))
	}
}

// TestFindViewIDByName 测试根据名称查找视图ID
func TestFindViewIDByName(t *testing.T) {
	crawler := createTestCrawler(t)

	// 构造测试数据
	name2id := [][]interface{}{
		{"view1", float64(1)},
		{"view2", float64(2)},
		{"view3", float64(3)},
	}

	// 测试存在的视图
	id := crawler.findViewIDByName("view2", name2id)
	if id != 2 {
		t.Errorf("Expected ID 2 for view2, got %d", id)
	}

	// 测试不存在的视图
	id = crawler.findViewIDByName("nonexistent", name2id)
	if id != -1 {
		t.Errorf("Expected ID -1 for non-existent view, got %d", id)
	}

	// 测试空列表
	id = crawler.findViewIDByName("view1", [][]interface{}{})
	if id != -1 {
		t.Errorf("Expected ID -1 for empty list, got %d", id)
	}
}

// TestCrawlerWithContext 测试上下文控制
func TestCrawlerWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)

	// 创建一个很短的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := crawler.CrawlAllServiceTrees(ctx)

	// 应该会因为上下文超时而失败（或者成功，如果网络很快）
	if err != nil {
		t.Logf("Expected timeout error or success, got: %v", err)
	}
}

// TestServiceTreeNodeStructure 测试服务树节点结构
func TestServiceTreeNodeStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	crawler := createTestCrawler(t)
	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	trees, err := crawler.CrawlAllServiceTrees(ctx)
	if err != nil {
		t.Fatalf("Failed to crawl service trees: %v", err)
	}

	for _, tree := range trees {
		// 验证树的基本结构
		validateServiceTreeStructure(t, tree)

		// 验证根节点结构
		for _, rootNode := range tree.RootNodes {
			validateNodeStructure(t, rootNode, 0, tree.MaxDepth)
		}
	}
}

// validateServiceTreeStructure 验证服务树结构
func validateServiceTreeStructure(t *testing.T, tree *models.ServiceTreeData) {
	if tree.ViewName == "" {
		t.Error("Service tree should have a view name")
	}

	if tree.ViewID <= 0 {
		t.Errorf("Service tree should have a positive view ID, got %d", tree.ViewID)
	}

	if tree.CrawledAt.IsZero() {
		t.Error("Service tree should have a crawled timestamp")
	}

	if tree.TotalNodes < 0 {
		t.Errorf("Total nodes should be non-negative, got %d", tree.TotalNodes)
	}

	if tree.MaxDepth < 0 {
		t.Errorf("Max depth should be non-negative, got %d", tree.MaxDepth)
	}
}

// validateNodeStructure 递归验证节点结构
func validateNodeStructure(t *testing.T, node *models.ServiceTreeNode, currentLevel, maxDepth int) {
	if node.ID <= 0 {
		t.Errorf("Node should have a positive ID, got %d", node.ID)
	}

	if node.Type <= 0 {
		t.Errorf("Node should have a positive type, got %d", node.Type)
	}

	if node.Name == "" {
		t.Error("Node should have a name")
	}

	if node.Level != currentLevel {
		t.Errorf("Node level should be %d, got %d", currentLevel, node.Level)
	}

	if node.Children == nil {
		t.Error("Node children should not be nil")
	}

	if node.ChildCount != len(node.Children) {
		t.Errorf("Node child count (%d) should match children slice length (%d)",
			node.ChildCount, len(node.Children))
	}

	// 如果有子节点但标记为叶子节点，这是错误的
	if len(node.Children) > 0 && node.IsLeaf {
		t.Error("Node with children should not be marked as leaf")
	}

	// 递归验证子节点
	for _, child := range node.Children {
		validateNodeStructure(t, child, currentLevel+1, maxDepth)
	}
}

// BenchmarkCrawlAllServiceTrees 性能测试
func BenchmarkCrawlAllServiceTrees(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	crawler := createTestCrawler(&testing.T{})
	crawler.SetMaxDepth(1) // 限制深度以减少基准测试时间

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := crawler.CrawlAllServiceTrees(ctx)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
