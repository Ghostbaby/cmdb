package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cmdb-crawler/internal/models"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ExportFormat 导出格式枚举
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatYAML ExportFormat = "yaml"
	FormatCSV  ExportFormat = "csv"
)

// Exporter 数据导出器
type Exporter struct {
	logger      *zap.Logger
	format      ExportFormat
	prettyPrint bool
}

// NewExporter 创建数据导出器
func NewExporter(format string, prettyPrint bool, logger *zap.Logger) *Exporter {
	return &Exporter{
		logger:      logger,
		format:      ExportFormat(strings.ToLower(format)),
		prettyPrint: prettyPrint,
	}
}

// ExportServiceTrees 导出服务树数据
func (e *Exporter) ExportServiceTrees(data []*models.ServiceTreeData, outputPath string) error {
	e.logger.Info("Exporting service trees",
		zap.String("format", string(e.format)),
		zap.String("output_path", outputPath),
		zap.Int("tree_count", len(data)))

	// 确保输出目录存在
	if err := e.ensureOutputDir(outputPath); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch e.format {
	case FormatJSON:
		return e.exportJSON(data, outputPath)
	case FormatYAML:
		return e.exportYAML(data, outputPath)
	case FormatCSV:
		return e.exportCSV(data, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", e.format)
	}
}

// exportJSON 导出为JSON格式
func (e *Exporter) exportJSON(data []*models.ServiceTreeData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if e.prettyPrint {
		encoder.SetIndent("", "  ")
	}

	// 添加元数据
	export := struct {
		Metadata     ExportMetadata            `json:"metadata"`
		ServiceTrees []*models.ServiceTreeData `json:"service_trees"`
	}{
		Metadata: ExportMetadata{
			ExportedAt: time.Now(),
			Format:     string(e.format),
			Version:    "1.0",
			TreeCount:  len(data),
			TotalNodes: e.countTotalNodes(data),
		},
		ServiceTrees: data,
	}

	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	e.logger.Info("Successfully exported JSON", zap.String("file", outputPath))
	return nil
}

// exportYAML 导出为YAML格式
func (e *Exporter) exportYAML(data []*models.ServiceTreeData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer file.Close()

	// 添加元数据
	export := struct {
		Metadata     ExportMetadata            `json:"metadata"`
		ServiceTrees []*models.ServiceTreeData `json:"service_trees"`
	}{
		Metadata: ExportMetadata{
			ExportedAt: time.Now(),
			Format:     string(e.format),
			Version:    "1.0",
			TreeCount:  len(data),
			TotalNodes: e.countTotalNodes(data),
		},
		ServiceTrees: data,
	}

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	e.logger.Info("Successfully exported YAML", zap.String("file", outputPath))
	return nil
}

// exportCSV 导出为CSV格式
func (e *Exporter) exportCSV(data []*models.ServiceTreeData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头部
	headers := []string{
		"view_name", "view_id", "node_id", "node_type", "node_type_name",
		"node_name", "node_path", "level", "is_leaf", "child_count", "parent_id",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// 遍历所有服务树
	for _, tree := range data {
		for _, rootNode := range tree.RootNodes {
			e.writeNodeToCSV(writer, tree.ViewName, tree.ViewID, rootNode, 0)
		}
	}

	e.logger.Info("Successfully exported CSV", zap.String("file", outputPath))
	return nil
}

// writeNodeToCSV 递归写入节点到CSV
func (e *Exporter) writeNodeToCSV(writer *csv.Writer, viewName string, viewID int,
	node *models.ServiceTreeNode, parentID int) {

	record := []string{
		viewName,
		fmt.Sprintf("%d", viewID),
		fmt.Sprintf("%d", node.ID),
		fmt.Sprintf("%d", node.Type),
		node.TypeName,
		node.Name,
		node.BuildTreePath(),
		fmt.Sprintf("%d", node.Level),
		fmt.Sprintf("%t", node.IsLeaf),
		fmt.Sprintf("%d", node.ChildCount),
		fmt.Sprintf("%d", parentID),
	}

	writer.Write(record)

	// 递归写入子节点
	for _, child := range node.Children {
		e.writeNodeToCSV(writer, viewName, viewID, child, node.ID)
	}
}

// ensureOutputDir 确保输出目录存在
func (e *Exporter) ensureOutputDir(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir == "." || dir == "/" {
		return nil
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		e.logger.Info("Created output directory", zap.String("dir", dir))
	}

	return nil
}

// countTotalNodes 统计总节点数
func (e *Exporter) countTotalNodes(data []*models.ServiceTreeData) int {
	total := 0
	for _, tree := range data {
		total += tree.TotalNodes
	}
	return total
}

// ExportMetadata 导出元数据
type ExportMetadata struct {
	ExportedAt time.Time `json:"exported_at" yaml:"exported_at"`
	Format     string    `json:"format" yaml:"format"`
	Version    string    `json:"version" yaml:"version"`
	TreeCount  int       `json:"tree_count" yaml:"tree_count"`
	TotalNodes int       `json:"total_nodes" yaml:"total_nodes"`
}

// ExportSingleTree 导出单个服务树
func (e *Exporter) ExportSingleTree(tree *models.ServiceTreeData, outputPath string) error {
	return e.ExportServiceTrees([]*models.ServiceTreeData{tree}, outputPath)
}

// ExportSummary 导出服务树摘要信息
func (e *Exporter) ExportSummary(data []*models.ServiceTreeData, outputPath string) error {
	summaries := make([]ServiceTreeSummary, len(data))

	for i, tree := range data {
		summaries[i] = ServiceTreeSummary{
			ViewName:   tree.ViewName,
			ViewID:     tree.ViewID,
			RootCount:  len(tree.RootNodes),
			TotalNodes: tree.TotalNodes,
			MaxDepth:   tree.MaxDepth,
			CrawledAt:  tree.CrawledAt,
			IsPublic:   tree.Config.IsPublic,
			LeafTypes:  e.getLeafTypeNames(tree),
		}
	}

	// 创建摘要导出结构
	export := struct {
		Metadata ExportMetadata       `json:"metadata"`
		Summary  []ServiceTreeSummary `json:"summary"`
	}{
		Metadata: ExportMetadata{
			ExportedAt: time.Now(),
			Format:     string(e.format),
			Version:    "1.0",
			TreeCount:  len(data),
			TotalNodes: e.countTotalNodes(data),
		},
		Summary: summaries,
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	switch e.format {
	case FormatJSON:
		encoder := json.NewEncoder(file)
		if e.prettyPrint {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(export)
	case FormatYAML:
		encoder := yaml.NewEncoder(file)
		defer encoder.Close()
		return encoder.Encode(export)
	default:
		return fmt.Errorf("unsupported format for summary: %s", e.format)
	}
}

// ServiceTreeSummary 服务树摘要
type ServiceTreeSummary struct {
	ViewName   string    `json:"view_name" yaml:"view_name"`
	ViewID     int       `json:"view_id" yaml:"view_id"`
	RootCount  int       `json:"root_count" yaml:"root_count"`
	TotalNodes int       `json:"total_nodes" yaml:"total_nodes"`
	MaxDepth   int       `json:"max_depth" yaml:"max_depth"`
	CrawledAt  time.Time `json:"crawled_at" yaml:"crawled_at"`
	IsPublic   bool      `json:"is_public" yaml:"is_public"`
	LeafTypes  []string  `json:"leaf_types" yaml:"leaf_types"`
}

// getLeafTypeNames 获取叶子节点类型名称
func (e *Exporter) getLeafTypeNames(tree *models.ServiceTreeData) []string {
	var leafTypes []string
	for _, showType := range tree.Config.ShowTypes {
		leafTypes = append(leafTypes, showType.Name)
	}
	return leafTypes
}

// GenerateFileName 生成文件名
func (e *Exporter) GenerateFileName(prefix string, timestamp bool) string {
	var filename string

	if timestamp {
		timeStr := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("%s_%s", prefix, timeStr)
	} else {
		filename = prefix
	}

	switch e.format {
	case FormatJSON:
		return filename + ".json"
	case FormatYAML:
		return filename + ".yaml"
	case FormatCSV:
		return filename + ".csv"
	default:
		return filename + ".txt"
	}
}
