package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cmdb-crawler/internal/client"
	"cmdb-crawler/internal/crawler"
	"cmdb-crawler/internal/models"
	"cmdb-crawler/internal/output"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	targetViews  []string
	outputPath   string
	outputFormat string
	maxDepth     int
	maxWorkers   int
	includeStats bool
	prettyPrint  bool
	summaryOnly  bool
)

// crawlCmd 爬取命令
var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "爬取服务树数据",
	Long: `从CMDB系统中爬取服务树数据

该命令会连接到CMDB系统，获取服务树视图配置，然后递归爬取所有节点数据。
支持并发爬取和多种输出格式。

示例：
  # 爬取所有服务树
  cmdb-crawler crawl

  # 爬取指定的服务树
  cmdb-crawler crawl --views "产品服务树,运维服务树"

  # 输出为CSV格式
  cmdb-crawler crawl --format csv --output ./data/trees.csv

  # 限制爬取深度为3层
  cmdb-crawler crawl --max-depth 3

  # 只输出摘要信息
  cmdb-crawler crawl --summary-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCrawl(cmd)
	},
}

func init() {
	rootCmd.AddCommand(crawlCmd)

	// 命令标志
	crawlCmd.Flags().StringSliceVar(&targetViews, "views", []string{}, "指定要爬取的服务树视图名称（逗号分隔）")
	crawlCmd.Flags().StringVarP(&outputPath, "output", "o", "", "输出文件路径")
	crawlCmd.Flags().StringVarP(&outputFormat, "format", "f", "", "输出格式 (json, yaml, csv)")
	crawlCmd.Flags().IntVar(&maxDepth, "max-depth", -1, "最大爬取深度 (-1表示无限制)")
	crawlCmd.Flags().IntVar(&maxWorkers, "max-workers", 0, "最大并发数")
	crawlCmd.Flags().BoolVar(&includeStats, "include-stats", true, "是否包含统计信息")
	crawlCmd.Flags().BoolVar(&prettyPrint, "pretty", false, "是否美化输出格式")
	crawlCmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "只输出摘要信息")
}

// runCrawl 执行爬取操作
func runCrawl(cmd *cobra.Command) error {
	logger := GetLogger()
	config := GetConfig()

	// 合并命令行参数和配置文件
	mergeFlags(config, cmd)

	logger.Info("开始爬取服务树数据",
		zap.String("cmdb_url", config.CMDB.BaseURL),
		zap.Strings("target_views", config.Crawler.ServiceTree.TargetViews),
		zap.String("output_format", config.Output.Format),
		zap.String("output_path", config.Output.FilePath))

	// 创建CMDB客户端
	cmdbClient := client.NewCMDBClient(config.CMDB.BaseURL, config.CMDB.APIVersion, logger)

	// 设置API Key认证
	if config.CMDB.Auth.APIKey == "" || config.CMDB.Auth.APISecret == "" {
		logger.Fatal("API Key和Secret不能为空，请在配置文件中设置")
	}

	logger.Info("Using API Key authentication")
	cmdbClient.SetAPICredentials(config.CMDB.Auth.APIKey, config.CMDB.Auth.APISecret)

	// 设置请求配置
	cmdbClient.SetTimeout(config.CMDB.Request.Timeout).
		SetRetry(config.CMDB.Request.RetryCount, config.CMDB.Request.RetryWaitTime)

	// 创建爬取器
	serviceCrawler := crawler.NewServiceTreeCrawler(cmdbClient, logger)
	serviceCrawler.SetMaxDepth(config.Crawler.ServiceTree.MaxDepth).
		SetPageSize(config.Crawler.ServiceTree.PageSize).
		SetMaxWorkers(config.Crawler.Concurrency.MaxWorkers).
		SetIncludeStats(config.Crawler.ServiceTree.IncludeStatistics).
		SetRequestInterval(config.Crawler.Concurrency.RequestInterval)

	// 执行爬取
	ctx := context.Background()
	var treeData []*models.ServiceTreeData
	var err error

	if len(config.Crawler.ServiceTree.TargetViews) > 0 {
		logger.Info("爬取指定的服务树视图",
			zap.Strings("views", config.Crawler.ServiceTree.TargetViews))
		treeData, err = serviceCrawler.CrawlSpecificViews(ctx, config.Crawler.ServiceTree.TargetViews)
	} else {
		logger.Info("爬取所有服务树视图")
		treeData, err = serviceCrawler.CrawlAllServiceTrees(ctx)
	}

	if err != nil {
		logger.Error("爬取失败", zap.Error(err))
		return fmt.Errorf("爬取服务树数据失败: %w", err)
	}

	if len(treeData) == 0 {
		logger.Warn("未找到任何服务树数据")
		fmt.Println("警告: 未找到任何服务树数据")
		return nil
	}

	// 输出结果
	if err := exportResults(treeData, config, logger); err != nil {
		logger.Error("导出结果失败", zap.Error(err))
		return fmt.Errorf("导出结果失败: %w", err)
	}

	// 输出统计信息
	printSummary(treeData, logger)

	return nil
}

// mergeFlags 合并命令行参数和配置文件
func mergeFlags(config *Config, cmd *cobra.Command) {
	// 目标视图
	if len(targetViews) > 0 {
		config.Crawler.ServiceTree.TargetViews = targetViews
	}

	// 输出路径
	if outputPath != "" {
		config.Output.FilePath = outputPath
	}

	// 输出格式
	if outputFormat != "" {
		config.Output.Format = outputFormat
	}

	// 最大深度
	if maxDepth != -1 {
		config.Crawler.ServiceTree.MaxDepth = maxDepth
	}

	// 最大并发数
	if maxWorkers > 0 {
		config.Crawler.Concurrency.MaxWorkers = maxWorkers
	}

	// 统计信息
	if cmd.Flags().Changed("include-stats") {
		config.Crawler.ServiceTree.IncludeStatistics = includeStats
	}

	// 美化输出
	if cmd.Flags().Changed("pretty") {
		config.Output.PrettyPrint = prettyPrint
	}
}

// exportResults 导出结果
func exportResults(treeData []*models.ServiceTreeData, config *Config, logger *zap.Logger) error {
	// 创建导出器
	exporter := output.NewExporter(config.Output.Format, config.Output.PrettyPrint, logger)

	// 生成输出文件路径
	outputFile := config.Output.FilePath
	if outputFile == "" {
		filename := exporter.GenerateFileName("service_tree_data", true)
		outputFile = filepath.Join("./output", filename)
	}

	// 导出数据
	if summaryOnly {
		// 只导出摘要
		summaryFile := strings.Replace(outputFile, filepath.Ext(outputFile), "_summary"+filepath.Ext(outputFile), 1)
		if err := exporter.ExportSummary(treeData, summaryFile); err != nil {
			return err
		}
		fmt.Printf("摘要信息已导出到: %s\n", summaryFile)
	} else {
		// 导出完整数据
		if err := exporter.ExportServiceTrees(treeData, outputFile); err != nil {
			return err
		}
		fmt.Printf("数据已导出到: %s\n", outputFile)

		// 同时生成摘要文件
		summaryFile := strings.Replace(outputFile, filepath.Ext(outputFile), "_summary"+filepath.Ext(outputFile), 1)
		if err := exporter.ExportSummary(treeData, summaryFile); err != nil {
			logger.Warn("生成摘要文件失败", zap.Error(err))
		} else {
			fmt.Printf("摘要信息已导出到: %s\n", summaryFile)
		}
	}

	return nil
}

// printSummary 打印统计摘要
func printSummary(treeData []*models.ServiceTreeData, logger *zap.Logger) {
	totalTrees := len(treeData)
	totalNodes := 0
	maxDepth := 0

	fmt.Println("\n=== 爬取结果摘要 ===")
	fmt.Printf("服务树总数: %d\n", totalTrees)

	for _, tree := range treeData {
		totalNodes += tree.TotalNodes
		if tree.MaxDepth > maxDepth {
			maxDepth = tree.MaxDepth
		}

		fmt.Printf("\n服务树: %s (ID: %d)\n", tree.ViewName, tree.ViewID)
		fmt.Printf("  根节点数: %d\n", len(tree.RootNodes))
		fmt.Printf("  总节点数: %d\n", tree.TotalNodes)
		fmt.Printf("  最大深度: %d\n", tree.MaxDepth)
		fmt.Printf("  是否公开: %t\n", tree.Config.IsPublic)
		fmt.Printf("  爬取时间: %s\n", tree.CrawledAt.Format("2006-01-02 15:04:05"))

		// 显示叶子节点类型
		if len(tree.Config.ShowTypes) > 0 {
			fmt.Printf("  叶子节点类型: ")
			for i, showType := range tree.Config.ShowTypes {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", showType.Alias)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n总计节点数: %d\n", totalNodes)
	fmt.Printf("最大深度: %d\n", maxDepth)
	fmt.Println("==================")

	logger.Info("爬取完成",
		zap.Int("total_trees", totalTrees),
		zap.Int("total_nodes", totalNodes),
		zap.Int("max_depth", maxDepth))
}
