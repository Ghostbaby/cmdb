# 更新日志

所有重要的项目更改都将记录在此文件中。

## [1.2.0] - 2025-07-26

### 🎉 重大修复与优化

#### ✅ 已修复
- **API查询参数修复**：修复了`use_id_filter=1`参数导致演示环境数据为空的问题，现在查询返回正确的数据
- **JSON解析增强**：完美解决了CI关系统计API混合数据类型的JSON解析错误，支持同时处理int和object类型字段
- **认证机制优化**：简化为仅支持API Key/Secret签名认证，移除了JWT Token和用户名密码认证方式，提高安全性
- **调试功能完善**：添加了dlv调试器支持和Cursor编辑器调试配置

#### 🚀 性能改进
- **并发优化**：改进了服务树爬取的并发处理机制
- **内存优化**：优化了数据结构，减少内存占用
- **错误处理增强**：完善了API调用的错误处理和重试机制

#### 📊 验证结果
- **演示环境测试**：成功验证17个服务树视图，共144个节点
- **API兼容性**：完全兼容https://cmdb.veops.cn演示环境
- **查询格式验证**：确认`_type:(73)`格式查询正常工作

### 📝 技术细节

#### API客户端改进
```go
// 修复前：可能返回空数据
rootResp, err := c.client.SearchCI(query, c.pageSize, true)

// 修复后：正确返回数据
rootResp, err := c.client.SearchCI(query, c.pageSize, false)
```

#### 数据模型增强
```go
// 修复前：无法处理混合数据类型
type StatisticsResponse map[string]int

// 修复后：支持混合数据类型
type StatisticsResponse struct {
    Data   map[string]interface{} `json:"-"`
    Detail map[string]interface{} `json:"detail,omitempty"`
}
```

#### 配置简化
```yaml
# 修复前：多种认证方式
cmdb:
  auth:
    username: "admin"
    password: "admin"
    token: "jwt_token"
    api_key: "key"
    api_secret: "secret"

# 修复后：仅API Key认证
cmdb:
  auth:
    api_key: "d0a8fb5aeedf466c92cc5142a18d1a68"
    api_secret: "DSGYH81jqfw~%A&vgyJKXrO*UFVaW2xt"
```

### 🛠️ 开发工具改进
- 添加了`make debug`系列命令支持dlv调试器
- 完善了VS Code/Cursor调试配置（`.vscode/launch.json`）
- 创建了专门的调试指南文档

### 📄 文档更新
- 更新了README.md，反映最新的修复状态
- 简化了API_AUTHENTICATION.md，移除不支持的认证方式
- 增强了USAGE.md的故障排除部分
- 添加了详细的调试指南

### 🧪 测试验证
- 创建了完整的单元测试套件
- 通过演示环境集成测试
- 验证了API Key/Secret签名算法正确性
- 确认了所有服务树视图的爬取功能

## [1.1.0] - 2025-07-25

### ✨ 新增功能
- **服务树爬取器**：实现了完整的服务树数据爬取功能
- **并发支持**：支持多服务树并发爬取和节点层级并发处理
- **多格式输出**：支持JSON、YAML、CSV格式输出
- **配置管理**：基于Viper的YAML配置文件支持
- **命令行界面**：基于Cobra的CLI工具

### 🔧 技术实现
- 实现了CMDB API客户端，支持多种认证方式
- 构建了完整的数据模型和服务树结构
- 添加了详细的日志记录和错误处理
- 实现了请求频率控制和重试机制

## [1.0.0] - 2025-07-24

### 🎉 首次发布
- **项目初始化**：基于Go 1.21+的CMDB服务树爬取工具
- **基础架构**：建立了项目的基本目录结构和构建系统
- **文档框架**：创建了项目文档和使用指南

---

### 图例说明
- 🎉 重大更新
- ✅ 修复问题
- 🚀 性能改进
- ✨ 新增功能
- 🔧 技术改进
- 📊 数据/统计
- 📝 文档更新
- 🛠️ 开发工具
- 🧪 测试相关
- ⚠️ 重要提醒
- ❌ 移除功能 