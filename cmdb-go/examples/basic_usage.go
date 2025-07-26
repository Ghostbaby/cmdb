package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cmdb-crawler/internal/client"
	"cmdb-crawler/internal/crawler"
	"cmdb-crawler/internal/models"
	"cmdb-crawler/internal/output"

	"go.uber.org/zap"
)

// 基本使用示例
func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("初始化日志失败:", err)
	}
	defer logger.Sync()

	// CMDB配置
	baseURL := "https://cmdb.veops.cn"
	apiVersion := "api/v0.1"
	apiKey := "your_api_key_here"
	apiSecret := "your_api_secret_here"

	// 创建CMDB客户端
	cmdbClient := client.NewCMDBClient(baseURL, apiVersion, logger)
	cmdbClient.SetAPICredentials(apiKey, apiSecret)
	cmdbClient.SetTimeout(30 * time.Second)
	cmdbClient.SetRetry(3, 1*time.Second)

	// 创建服务树爬取器
	serviceCrawler := crawler.NewServiceTreeCrawler(cmdbClient, logger)
	serviceCrawler.SetMaxDepth(5). // 最大深度5层
					SetPageSize(1000).                         // 每页1000条
					SetMaxWorkers(10).                         // 最大并发10
					SetIncludeStats(true).                     // 包含统计信息
					SetRequestInterval(100 * time.Millisecond) // 请求间隔100ms

	fmt.Println("开始爬取所有服务树...")

	// 爬取所有服务树
	ctx := context.Background()
	treeData, err := serviceCrawler.CrawlAllServiceTrees(ctx)
	if err != nil {
		logger.Fatal("爬取失败", zap.Error(err))
	}

	if len(treeData) == 0 {
		fmt.Println("未找到任何服务树数据")
		return
	}

	fmt.Printf("成功爬取 %d 个服务树\n", len(treeData))

	// 创建导出器并导出为JSON
	exporter := output.NewExporter("json", true, logger)

	// 导出完整数据
	if err := exporter.ExportServiceTrees(treeData, "./output/service_trees.json"); err != nil {
		logger.Fatal("导出失败", zap.Error(err))
	}

	// 导出摘要
	if err := exporter.ExportSummary(treeData, "./output/service_trees_summary.json"); err != nil {
		logger.Fatal("导出摘要失败", zap.Error(err))
	}

	// 打印统计信息
	printStatistics(treeData)

	fmt.Println("爬取完成！")
}

// printStatistics 打印统计信息
func printStatistics(treeData []*models.ServiceTreeData) {
	fmt.Println("\n=== 统计信息 ===")

	totalNodes := 0
	maxDepth := 0

	for _, tree := range treeData {
		totalNodes += tree.TotalNodes
		if tree.MaxDepth > maxDepth {
			maxDepth = tree.MaxDepth
		}

		fmt.Printf("服务树: %s\n", tree.ViewName)
		fmt.Printf("  根节点: %d 个\n", len(tree.RootNodes))
		fmt.Printf("  总节点: %d 个\n", tree.TotalNodes)
		fmt.Printf("  最大深度: %d 层\n", tree.MaxDepth)
		fmt.Printf("  爬取时间: %s\n", tree.CrawledAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	fmt.Printf("总计: %d 个服务树, %d 个节点, 最大深度 %d 层\n",
		len(treeData), totalNodes, maxDepth)
	fmt.Println("==================")
}
