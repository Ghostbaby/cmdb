package models

import (
	"encoding/json"
	"time"
)

// RelationViewResponse 服务树视图API响应
type RelationViewResponse struct {
	Views   map[string]ServiceTreeView `json:"views"`
	ID2Type map[string]CIType          `json:"id2type"`
	Name2ID [][]interface{}            `json:"name2id"`
}

// ServiceTreeView 服务树视图配置
type ServiceTreeView struct {
	Topo             [][]int             `json:"topo"`
	TopoFlatten      []int               `json:"topo_flatten"`
	Leaf             []int               `json:"leaf"`
	Leaf2ShowTypes   map[string][]int    `json:"leaf2show_types"`
	Node2ShowTypes   map[string][]CIType `json:"node2show_types"`
	Level2Constraint map[string]string   `json:"level2constraint"`
	Option           ServiceTreeOption   `json:"option"`
	IsPublic         bool                `json:"is_public"`
	ShowTypes        []CIType            `json:"show_types"`
}

// ServiceTreeOption 服务树选项配置
type ServiceTreeOption struct {
	IsShowLeafNode bool `json:"is_show_leaf_node"`
	IsShowTreeNode bool `json:"is_show_tree_node"`
	Sort           int  `json:"sort"`
	IsPublic       bool `json:"is_public"`
}

// CIType CI类型信息
type CIType struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	UniqueName string `json:"unique_name,omitempty"`
	ShowName   string `json:"show_name,omitempty"`
}

// CIInstance CI实例
type CIInstance struct {
	ID       int                    `json:"_id"`
	Type     int                    `json:"_type"`
	Name     string                 `json:"name,omitempty"`
	Unique   string                 `json:"unique,omitempty"`
	Attrs    map[string]interface{} `json:"-"`
	TypeName string                 `json:"type_name,omitempty"`
}

// CISearchResponse CI搜索API响应
type CISearchResponse struct {
	Result   []CIInstance `json:"result"`
	NumFound int          `json:"numfound"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
}

// CIRelationSearchResponse CI关系搜索API响应
type CIRelationSearchResponse struct {
	Result   []CIInstance `json:"result"`
	NumFound int          `json:"numfound"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	Counter  interface{}  `json:"counter"`
	Facet    interface{}  `json:"facet"`
}

// StatisticsResponse 统计API响应
type StatisticsResponse struct {
	// 根节点统计数据 (动态字段)
	Data map[string]interface{} `json:"-"`
	// 详细信息
	Detail map[string]interface{} `json:"detail,omitempty"`
}

// UnmarshalJSON 自定义JSON反序列化
func (s *StatisticsResponse) UnmarshalJSON(data []byte) error {
	// 先解析为通用map
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Data = make(map[string]interface{})

	// 分离detail字段和其他字段
	for key, value := range raw {
		if key == "detail" {
			if detailMap, ok := value.(map[string]interface{}); ok {
				s.Detail = detailMap
			}
		} else {
			s.Data[key] = value
		}
	}

	return nil
}

// GetCount 获取指定根节点的统计数量
func (s *StatisticsResponse) GetCount(rootID string) int {
	if s.Data == nil {
		return 0
	}

	if count, exists := s.Data[rootID]; exists {
		if intCount, ok := count.(float64); ok {
			return int(intCount)
		}
		if intCount, ok := count.(int); ok {
			return intCount
		}
	}

	return 0
}

// ServiceTreeNode 服务树节点
type ServiceTreeNode struct {
	ID         int                    `json:"id"`
	Type       int                    `json:"type"`
	TypeName   string                 `json:"type_name"`
	Name       string                 `json:"name"`
	Path       string                 `json:"path"`
	Level      int                    `json:"level"`
	Children   []*ServiceTreeNode     `json:"children,omitempty"`
	ChildCount int                    `json:"child_count"`
	IsLeaf     bool                   `json:"is_leaf"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Statistics map[string]int         `json:"statistics,omitempty"`
}

// ServiceTreeData 完整的服务树数据
type ServiceTreeData struct {
	ViewName   string             `json:"view_name"`
	ViewID     int                `json:"view_id"`
	Config     ServiceTreeView    `json:"config"`
	RootNodes  []*ServiceTreeNode `json:"root_nodes"`
	TotalNodes int                `json:"total_nodes"`
	MaxDepth   int                `json:"max_depth"`
	CrawledAt  time.Time          `json:"crawled_at"`
}

// UnmarshalJSON 自定义JSON反序列化，处理动态属性
func (ci *CIInstance) UnmarshalJSON(data []byte) error {
	type Alias CIInstance
	alias := &struct {
		*Alias
	}{
		Alias: (*Alias)(ci),
	}

	if err := json.Unmarshal(data, alias); err != nil {
		return err
	}

	// 解析所有其他属性到Attrs字段
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	ci.Attrs = make(map[string]interface{})
	for k, v := range raw {
		switch k {
		case "_id", "_type", "name", "unique", "type_name":
			// 这些是结构化字段，跳过
			continue
		default:
			ci.Attrs[k] = v
		}
	}

	return nil
}

// GetDisplayName 获取显示名称
func (ci *CIInstance) GetDisplayName() string {
	if ci.Name != "" {
		return ci.Name
	}
	if unique, ok := ci.Attrs[ci.Unique]; ok {
		if str, ok := unique.(string); ok {
			return str
		}
	}
	return ""
}

// BuildTreePath 构建树路径字符串
func (node *ServiceTreeNode) BuildTreePath() string {
	if node.Path == "" {
		return node.Name
	}
	return node.Path + " > " + node.Name
}

// AddChild 添加子节点
func (node *ServiceTreeNode) AddChild(child *ServiceTreeNode) {
	if node.Children == nil {
		node.Children = make([]*ServiceTreeNode, 0)
	}
	child.Path = node.BuildTreePath()
	child.Level = node.Level + 1
	node.Children = append(node.Children, child)
}

// GetAllDescendants 获取所有后代节点
func (node *ServiceTreeNode) GetAllDescendants() []*ServiceTreeNode {
	var descendants []*ServiceTreeNode

	for _, child := range node.Children {
		descendants = append(descendants, child)
		descendants = append(descendants, child.GetAllDescendants()...)
	}

	return descendants
}

// CountNodes 统计节点总数
func (data *ServiceTreeData) CountNodes() int {
	count := len(data.RootNodes)
	for _, root := range data.RootNodes {
		count += len(root.GetAllDescendants())
	}
	data.TotalNodes = count
	return count
}

// CalculateMaxDepth 计算最大深度
func (data *ServiceTreeData) CalculateMaxDepth() int {
	maxDepth := 0
	for _, root := range data.RootNodes {
		depth := calculateNodeDepth(root, 1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	data.MaxDepth = maxDepth
	return maxDepth
}

// calculateNodeDepth 递归计算节点深度
func calculateNodeDepth(node *ServiceTreeNode, currentDepth int) int {
	if len(node.Children) == 0 {
		return currentDepth
	}

	maxChildDepth := currentDepth
	for _, child := range node.Children {
		childDepth := calculateNodeDepth(child, currentDepth+1)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return maxChildDepth
}
