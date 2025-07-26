package client

import (
	"crypto/sha1"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cmdb-crawler/internal/models"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// CMDBClient CMDB API客户端
type CMDBClient struct {
	client     *resty.Client
	baseURL    string
	apiVersion string
	logger     *zap.Logger
	// API Key认证
	apiKey    string
	apiSecret string
}

// NewCMDBClient 创建CMDB客户端
func NewCMDBClient(baseURL, apiVersion string, logger *zap.Logger) *CMDBClient {
	client := resty.New()

	// 设置默认请求头
	client.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"User-Agent":   "CMDB-Crawler/1.0",
	})

	// 启用Cookie支持
	client.SetCookieJar(nil)

	return &CMDBClient{
		client:     client,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiVersion: apiVersion,
		logger:     logger,
	}
}

// SetAPICredentials 设置API Key认证
func (c *CMDBClient) SetAPICredentials(apiKey, apiSecret string) *CMDBClient {
	c.apiKey = apiKey
	c.apiSecret = apiSecret
	c.logger.Info("Set API key authentication", zap.String("api_key", apiKey))
	return c
}

// buildSignature 构建API签名
func (c *CMDBClient) buildSignature(urlPath string, params map[string]string) string {
	// 1. 收集除_key和_secret外的所有参数
	var keys []string
	for k := range params {
		if k != "_key" && k != "_secret" {
			keys = append(keys, k)
		}
	}

	// 2. 参数名排序
	sort.Strings(keys)

	// 3. 拼接参数值
	var values []string
	for _, k := range keys {
		values = append(values, params[k])
	}
	paramValues := strings.Join(values, "")

	// 4. 构建签名字符串：url_path + secret + 参数值
	signStr := urlPath + c.apiSecret + paramValues

	// 5. 计算SHA1
	h := sha1.New()
	h.Write([]byte(signStr))
	signature := fmt.Sprintf("%x", h.Sum(nil))

	c.logger.Debug("API signature",
		zap.String("url_path", urlPath),
		zap.String("sign_string", signStr),
		zap.String("signature", signature))

	return signature
}

// addAPIAuth 为请求添加API认证参数
func (c *CMDBClient) addAPIAuth(urlPath string, params map[string]string) map[string]string {
	if c.apiKey == "" || c.apiSecret == "" {
		c.logger.Error("API credentials not set")
		return params
	}

	if params == nil {
		params = make(map[string]string)
	}

	// 添加API Key
	params["_key"] = c.apiKey

	// 计算并添加签名
	params["_secret"] = c.buildSignature(urlPath, params)

	return params
}

// SetTimeout 设置请求超时
func (c *CMDBClient) SetTimeout(timeout time.Duration) *CMDBClient {
	c.client.SetTimeout(timeout)
	return c
}

// SetRetry 设置重试配置
func (c *CMDBClient) SetRetry(count int, waitTime time.Duration) *CMDBClient {
	c.client.SetRetryCount(count).SetRetryWaitTime(waitTime)
	return c
}

// buildURL 构建完整的API URL
func (c *CMDBClient) buildURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "/") {
		endpoint = endpoint[1:]
	}
	return fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, endpoint)
}

// getURLPath 从完整URL中提取路径部分
func (c *CMDBClient) getURLPath(fullURL string) string {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		c.logger.Error("Failed to parse URL", zap.String("url", fullURL), zap.Error(err))
		return ""
	}
	return parsedURL.Path
}

// GetRelationViews 获取服务树视图列表
func (c *CMDBClient) GetRelationViews() (*models.RelationViewResponse, error) {
	c.logger.Info("Fetching relation views")

	fullURL := c.buildURL("preference/relation/view")
	urlPath := c.getURLPath(fullURL)
	c.logger.Debug("Request URL", zap.String("url", fullURL), zap.String("path", urlPath))

	var response models.RelationViewResponse

	// 构建查询参数并添加API认证
	params := c.addAPIAuth(urlPath, nil)

	resp, err := c.client.R().
		SetQueryParams(params).
		SetResult(&response).
		Get(fullURL)

	if err != nil {
		c.logger.Error("Failed to get relation views", zap.Error(err))
		return nil, fmt.Errorf("failed to get relation views: %w", err)
	}

	c.logger.Debug("API Response",
		zap.Int("status", resp.StatusCode()),
		zap.String("body", string(resp.Body())))

	if resp.StatusCode() != 200 {
		c.logger.Error("API returned non-200 status",
			zap.Int("status", resp.StatusCode()),
			zap.String("body", string(resp.Body())))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("Successfully fetched relation views",
		zap.Int("view_count", len(response.Views)))

	return &response, nil
}

// SearchCI 搜索CI实例
func (c *CMDBClient) SearchCI(query string, count int, useIDFilter bool) (*models.CISearchResponse, error) {
	c.logger.Info("Searching CI instances",
		zap.String("query", query),
		zap.Int("count", count))

	var response models.CISearchResponse

	fullURL := c.buildURL("ci/s")
	urlPath := c.getURLPath(fullURL)

	// 构建查询参数
	params := map[string]string{
		"q":     query,
		"count": strconv.Itoa(count),
	}

	if useIDFilter {
		params["use_id_filter"] = "1"
	}

	// 添加API认证
	params = c.addAPIAuth(urlPath, params)

	resp, err := c.client.R().
		SetQueryParams(params).
		SetResult(&response).
		Get(fullURL)

	if err != nil {
		c.logger.Error("Failed to search CI instances", zap.Error(err))
		return nil, fmt.Errorf("failed to search CI instances: %w", err)
	}

	if resp.StatusCode() != 200 {
		c.logger.Error("API returned non-200 status",
			zap.Int("status", resp.StatusCode()),
			zap.String("body", string(resp.Body())))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("Successfully searched CI instances",
		zap.Int("found", response.NumFound),
		zap.Int("returned", len(response.Result)))

	return &response, nil
}

// SearchCIRelation 搜索CI关系
func (c *CMDBClient) SearchCIRelation(queryParams map[string]interface{}) (*models.CIRelationSearchResponse, error) {
	c.logger.Info("Searching CI relations", zap.Any("params", queryParams))

	var response models.CIRelationSearchResponse

	fullURL := c.buildURL("ci_relations/s")
	urlPath := c.getURLPath(fullURL)

	// 转换参数
	params := make(map[string]string)
	for k, v := range queryParams {
		switch val := v.(type) {
		case string:
			params[k] = val
		case int:
			params[k] = strconv.Itoa(val)
		case []int:
			strs := make([]string, len(val))
			for i, num := range val {
				strs[i] = strconv.Itoa(num)
			}
			params[k] = strings.Join(strs, ",")
		case []string:
			params[k] = strings.Join(val, ",")
		default:
			params[k] = fmt.Sprintf("%v", val)
		}
	}

	// 添加API认证
	params = c.addAPIAuth(urlPath, params)

	resp, err := c.client.R().
		SetQueryParams(params).
		SetResult(&response).
		Get(fullURL)

	if err != nil {
		c.logger.Error("Failed to search CI relations", zap.Error(err))
		return nil, fmt.Errorf("failed to search CI relations: %w", err)
	}

	if resp.StatusCode() != 200 {
		c.logger.Error("API returned non-200 status",
			zap.Int("status", resp.StatusCode()),
			zap.String("body", string(resp.Body())))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("Successfully searched CI relations",
		zap.Int("found", response.NumFound),
		zap.Int("returned", len(response.Result)))

	return &response, nil
}

// GetCIRelationStatistics 获取CI关系统计
func (c *CMDBClient) GetCIRelationStatistics(queryParams map[string]interface{}) (models.StatisticsResponse, error) {
	c.logger.Info("Getting CI relation statistics", zap.Any("params", queryParams))

	var response models.StatisticsResponse

	fullURL := c.buildURL("ci_relations/statistics")
	urlPath := c.getURLPath(fullURL)

	// 转换参数
	params := make(map[string]string)
	for k, v := range queryParams {
		switch val := v.(type) {
		case string:
			params[k] = val
		case int:
			params[k] = strconv.Itoa(val)
		case []int:
			strs := make([]string, len(val))
			for i, num := range val {
				strs[i] = strconv.Itoa(num)
			}
			params[k] = strings.Join(strs, ",")
		case []string:
			params[k] = strings.Join(val, ",")
		default:
			params[k] = fmt.Sprintf("%v", val)
		}
	}

	// 添加API认证
	params = c.addAPIAuth(urlPath, params)

	resp, err := c.client.R().
		SetQueryParams(params).
		SetResult(&response).
		Get(fullURL)

	if err != nil {
		c.logger.Error("Failed to get CI relation statistics", zap.Error(err))
		return models.StatisticsResponse{}, fmt.Errorf("failed to get CI relation statistics: %w", err)
	}

	if resp.StatusCode() != 200 {
		c.logger.Error("API returned non-200 status",
			zap.Int("status", resp.StatusCode()),
			zap.String("body", string(resp.Body())))
		return models.StatisticsResponse{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("Successfully got CI relation statistics",
		zap.Int("stat_count", len(response.Data)))

	return response, nil
}

// BuildCITypeQuery 构建CI类型查询字符串
func (c *CMDBClient) BuildCITypeQuery(typeIDs []int) string {
	if len(typeIDs) == 0 {
		return ""
	}

	strs := make([]string, len(typeIDs))
	for i, id := range typeIDs {
		strs[i] = strconv.Itoa(id)
	}

	return fmt.Sprintf("_type:(%s)", strings.Join(strs, ";"))
}

// ParseTreeKey 解析树节点Key
func (c *CMDBClient) ParseTreeKey(key string) ([]TreeKeySegment, error) {
	if key == "" {
		return nil, nil
	}

	segments := strings.Split(key, "@^@")
	result := make([]TreeKeySegment, len(segments))

	for i, segment := range segments {
		parts := strings.Split(segment, "%")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid tree key segment: %s", segment)
		}

		ciID, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid CI ID in segment: %s", parts[0])
		}

		typeID, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid type ID in segment: %s", parts[1])
		}

		result[i] = TreeKeySegment{
			CIID:   ciID,
			TypeID: typeID,
			Meta:   parts[2],
		}
	}

	return result, nil
}

// TreeKeySegment 树节点Key片段
type TreeKeySegment struct {
	CIID   int    `json:"ci_id"`
	TypeID int    `json:"type_id"`
	Meta   string `json:"meta"`
}

// BuildTreeKey 构建树节点Key
func (c *CMDBClient) BuildTreeKey(segments []TreeKeySegment) string {
	if len(segments) == 0 {
		return ""
	}

	parts := make([]string, len(segments))
	for i, seg := range segments {
		parts[i] = fmt.Sprintf("%d%%%d%%%s", seg.CIID, seg.TypeID, seg.Meta)
	}

	return strings.Join(parts, "@^@")
}

// ValidateResponse 验证API响应
func (c *CMDBClient) ValidateResponse(resp *resty.Response) error {
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}
	return nil
}
