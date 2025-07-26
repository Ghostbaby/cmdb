# CMDB 服务树数据爬取工具

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

一个高性能的CMDB服务树数据爬取工具，基于Golang开发，支持并发爬取和多种数据输出格式。

## 🎯 重要更新 - API认证方式

**通过分析[Veops CMDB API官方文档](https://veops.cn/docs/docs/cmdb/cmdb_api)，我们发现CMDB系统使用API Key/Secret签名认证！**

❌ **之前的误解**：从前端代码分析以为使用JWT Token认证  
✅ **正确方式**：API调用需要使用API Key/Secret签名认证  

**请查看 [API_AUTHENTICATION.md](API_AUTHENTICATION.md) 获取详细配置指南！**

## ✨ 特性

- 🔐 **多种认证方式**：API Key/Secret签名认证（推荐）、JWT Token、用户名密码
- 🚀 **高并发爬取**：支持配置最大并发数和请求间隔
- 📊 **多格式输出**：JSON、YAML、CSV格式支持
- 🎯 **智能过滤**：支持指定服务树视图、深度限制
- 📈 **统计信息**：详细的爬取统计和节点计数
- 🔄 **错误重试**：自动重试机制和详细错误日志
- ⚙️ **灵活配置**：YAML配置文件和命令行参数支持

## 🏗️ 技术架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   配置管理       │    │    API客户端      │    │   并发爬取器     │
│  (Viper+YAML)  │───▶│ (Resty+签名认证)  │───▶│ (Goroutines)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   数据模型       │    │    CMDB API      │    │   数据输出       │
│ (Go Structs)   │◀───│   (v0.1/v1)      │    │ (JSON/YAML/CSV)│
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 📋 API认证说明

### 🔑 API Key认证（推荐）
```yaml
cmdb:
  auth:
    api_key: "your_api_key_from_acl_system"
    api_secret: "your_api_secret_from_acl_system" 
```

### 📝 签名算法
严格按照官方文档实现：
```
SHA1(url_path + secret + 参数名排序后拼接的参数值)
```

### 🔄 备用认证
- JWT Token认证
- 用户名密码认证

## 🚀 快速开始

### 1. 获取API凭据

1. 登录您的CMDB系统
2. 进入**ACL系统**或**权限管理**
3. 查看您的`api key`和`secret`

### 2. 配置认证信息

编辑 `config/config.yaml`：
```yaml
cmdb:
  base_url: "https://your-cmdb-server.com"
  auth:
    api_key: "your_real_api_key"
    api_secret: "your_real_api_secret"
```

### 3. 运行爬取

```bash
# 构建项目
go build -o cmdb-crawler main.go

# 爬取所有服务树
./cmdb-crawler crawl --verbose

# 爬取指定服务树
./cmdb-crawler crawl --views "产品服务树,运维服务树" --format yaml

# 限制深度和并发
./cmdb-crawler crawl --max-depth 3 --max-workers 5
```

## 📊 输出示例

### JSON格式
```json
{
  "service_trees": [
    {
      "view_id": 1,
      "view_name": "产品服务树",
      "root_nodes": [
        {
          "ci_id": 100,
          "ci_type": "product_line",
          "display_name": "核心产品线",
          "children": [...]
        }
      ],
      "total_nodes": 1250,
      "max_depth": 4,
      "crawled_at": "2025-01-26T10:30:00Z"
    }
  ],
  "summary": {
    "total_trees": 3,
    "total_nodes": 5000,
    "crawl_duration": "2.5s"
  }
}
```

## 🔧 高级配置

### 完整配置示例
```yaml
cmdb:
  base_url: "https://cmdb.veops.cn"
  api_version: "api/v0.1"
  auth:
    api_key: "your_api_key"
    api_secret: "your_api_secret"
  request:
    timeout: 30s
    retry_count: 3
    retry_wait_time: 1s

crawler:
  service_tree:
    target_views: ["产品服务树"]
    max_depth: 5
    page_size: 1000
    include_statistics: true
  concurrency:
    max_workers: 10
    request_interval: 100ms

output:
  format: "json"
  file_path: "./output/service_trees.json"
  pretty_print: true

logging:
  level: "info"
  output: "console"
```

## 📚 API端点映射

| 功能 | API端点 | 实现方法 |
|------|---------|----------|
| 获取服务树视图 | `/api/v0.1/preference/relation/view` | `GetRelationViews()` |
| 搜索CI实例 | `/api/v0.1/ci/s` | `SearchCI()` |
| 查询CI关系 | `/api/v0.1/ci_relations/s` | `SearchCIRelation()` |
| 关系统计 | `/api/v0.1/ci_relations/statistics` | `GetCIRelationStatistics()` |

## 🎯 实际案例

### CI类型层级
```
产品线(39) → 产品(2) → 项目(3)
                  ↓
              环境(40) → K8S集群(41)
```

### 爬取流程
1. 获取服务树视图配置
2. 根据`topo`配置确定层级关系
3. 并发爬取每一层的CI实例
4. 构建完整的树形结构
5. 输出统计和数据文件

## 🔍 故障排除

### 401认证错误
```bash
{"level":"error","msg":"API returned status 401: unauthorized"}
```
**解决方案**：检查API Key和Secret是否正确

### 404端点错误
```bash
{"level":"error","msg":"404 Not Found"}
```
**解决方案**：检查API版本和端点路径

### 网络超时
```bash
{"level":"error","msg":"request timeout"}
```
**解决方案**：增加timeout配置或检查网络连接

## 🌟 成功输出示例

```bash
✅ 使用API Key认证
✅ 成功连接到 https://cmdb.veops.cn
✅ 获取到 3 个服务树视图
✅ 开始爬取: 产品服务树 (1/3)
✅ 爬取完成: 1250 个节点, 最大深度 4 层
✅ 数据已导出到: ./output/service_trees.json
✅ 摘要已导出到: ./output/service_trees_summary.json

=== 爬取结果摘要 ===
服务树总数: 3
总计节点数: 5000
最大深度: 5
爬取耗时: 3.2s
==================
```

## 📄 文档

- **[API认证配置指南](API_AUTHENTICATION.md)** - 详细的认证配置说明
- **[调试指南](DEBUG_GUIDE.md)** - dlv调试器使用指南和技巧
- **[Cursor调试指南](CURSOR_DEBUG_GUIDE.md)** - Cursor编辑器专用调试配置
- **[使用手册](USAGE.md)** - 完整的使用指南和示例
- **[官方API文档](https://veops.cn/docs/docs/cmdb/cmdb_api)** - Veops CMDB API参考

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📜 许可证

MIT License

## 🙏 致谢

- [Veops CMDB](https://veops.cn) - 提供优秀的CMDB系统和API文档
- [Go-Resty](https://github.com/go-resty/resty) - HTTP客户端库
- [Cobra](https://github.com/spf13/cobra) - CLI框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Zap](https://github.com/uber-go/zap) - 日志库 