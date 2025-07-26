package client

import (
	"testing"

	"go.uber.org/zap"
)

// TestBuildCITypeQuery 测试CI类型查询字符串构建
func TestBuildCITypeQuery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewCMDBClient("http://example.com", "api/v0.1", logger)

	tests := []struct {
		name     string
		typeIDs  []int
		expected string
	}{
		{
			name:     "空类型列表",
			typeIDs:  []int{},
			expected: "",
		},
		{
			name:     "单个类型ID",
			typeIDs:  []int{73},
			expected: "_type:(73)",
		},
		{
			name:     "多个类型ID",
			typeIDs:  []int{73, 74, 75},
			expected: "_type:(73;74;75)",
		},
		{
			name:     "两个类型ID",
			typeIDs:  []int{39, 40},
			expected: "_type:(39;40)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.BuildCITypeQuery(tt.typeIDs)
			if result != tt.expected {
				t.Errorf("BuildCITypeQuery(%v) = %q, expected %q", tt.typeIDs, result, tt.expected)
			}
		})
	}
}

// TestBuildCITypeQuery_KKKK_View 专门测试KKKK视图的情况
func TestBuildCITypeQuery_KKKK_View(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewCMDBClient("http://example.com", "api/v0.1", logger)

	// KKKK视图的根节点类型ID是73
	result := client.BuildCITypeQuery([]int{73})
	expected := "_type:(73)"

	if result != expected {
		t.Errorf("KKKK view query: got %q, expected %q", result, expected)
	}

	t.Logf("KKKK视图查询修复验证: %s", result)
}
