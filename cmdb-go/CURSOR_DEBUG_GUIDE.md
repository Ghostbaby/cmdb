# Cursor 调试配置指南

本文档介绍如何在Cursor编辑器中调试CMDB服务树数据爬取工具。

## 🎯 配置概览

我们为Cursor提供了7种不同的调试配置，满足各种调试需求：

### 1. **Launch Package** 
- 基础包调试配置
- 适用于调试单个文件或包

### 2. **Debug CMDB Crawler** ⭐
- 标准调试配置，使用 `crawl --verbose` 参数
- 最常用的调试配置

### 3. **Debug CMDB Crawler with Custom Args**
- 自定义参数调试，使用 `crawl --format yaml --pretty`
- 可根据需要修改参数

### 4. **Connect to Remote Debug Server** ⭐
- 连接到远程调试服务器（端口2345）
- 配合 `make debug` 使用

### 5. **Connect to Custom Debug Port**
- 连接到自定义端口（端口3456）
- 配合 `make debug DEBUG_PORT=3456` 使用

### 6. **Debug Specific View**
- 调试特定服务树视图
- 示例：调试"产品服务树"

### 7. **Debug Tests**
- 调试单元测试
- 自动发现并调试测试文件

## 🚀 使用方法

### 方式1：直接在Cursor中调试（推荐）

1. **打开调试面板**：
   - 按 `Cmd+Shift+D` (Mac) 或 `Ctrl+Shift+D` (Windows/Linux)
   - 或点击左侧活动栏的调试图标 🐛

2. **选择调试配置**：
   - 在调试面板顶部的下拉菜单中选择 "Debug CMDB Crawler"

3. **设置断点**：
   - 在代码行号左侧点击设置断点
   - 推荐在以下位置设置断点：
     ```go
     // main.go
     func main() {
         // 在这里设置断点
     }
     
     // internal/client/cmdb_client.go
     func (c *CMDBClient) SetAPICredentials(apiKey, apiSecret string) *CMDBClient {
         // 在这里设置断点
     }
     ```

4. **开始调试**：
   - 按 `F5` 或点击绿色播放按钮 ▶️
   - 程序会在断点处暂停

### 方式2：远程调试（高级用户）

1. **启动调试服务器**：
   ```bash
   make debug
   ```

2. **在Cursor中连接**：
   - 选择 "Connect to Remote Debug Server" 配置
   - 按 `F5` 开始连接
   - Cursor会连接到localhost:2345

3. **开始调试**：
   - 设置断点后，程序会在断点处暂停
   - 支持多个客户端同时连接

## 🎛️ 调试控制

### 调试工具栏
当调试启动后，会出现调试工具栏：

- **继续** (`F5`) - 继续执行到下一个断点
- **单步跳过** (`F10`) - 执行当前行，不进入函数内部
- **单步调试** (`F11`) - 执行当前行，进入函数内部
- **单步跳出** (`Shift+F11`) - 跳出当前函数
- **重启** (`Ctrl+Shift+F5`) - 重启调试会话
- **停止** (`Shift+F5`) - 停止调试

### 调试面板功能

#### 变量面板
- 查看局部变量、全局变量
- 可展开复杂数据结构
- 支持变量值的实时修改

#### 监视面板
- 添加监视表达式
- 实时查看表达式值
- 示例监视表达式：
  ```
  config.CMDB.Auth.APIKey
  len(response.Views)
  c.apiSecret
  ```

#### 调用堆栈面板
- 查看函数调用链
- 点击堆栈项可跳转到对应代码位置
- 在goroutine之间切换

#### 断点面板
- 管理所有断点
- 可以临时禁用/启用断点
- 设置条件断点

## 🔧 自定义调试配置

### 修改现有配置

如果需要调试不同的命令或参数，可以修改 `.vscode/launch.json`：

```json
{
    "name": "Debug My Custom Command",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/main.go",
    "args": ["crawl", "--views", "我的服务树", "--format", "csv"],
    "env": {},
    "showLog": true,
    "buildFlags": "-gcflags='all=-N -l'",
    "cwd": "${workspaceFolder}",
    "console": "integratedTerminal"
}
```

### 添加环境变量

```json
{
    "name": "Debug with Environment",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/main.go",
    "args": ["crawl"],
    "env": {
        "CMDB_AUTH_API_KEY": "your_test_key",
        "CMDB_BASE_URL": "https://test.cmdb.com"
    },
    "buildFlags": "-gcflags='all=-N -l'"
}
```

## 🎯 调试场景示例

### 场景1：调试API认证

1. **设置断点**：
   ```go
   // internal/client/cmdb_client.go:76
   func (c *CMDBClient) SetAPICredentials(apiKey, apiSecret string) *CMDBClient
   ```

2. **启动调试**：选择 "Debug CMDB Crawler" 并按 `F5`

3. **查看变量**：
   - 在变量面板查看 `apiKey` 和 `apiSecret`
   - 在监视面板添加 `c.apiKey` 和 `c.apiSecret`

4. **单步调试**：
   - 按 `F11` 进入 `buildSignature` 函数
   - 观察签名计算过程

### 场景2：调试服务树爬取

1. **设置断点**：
   ```go
   // internal/crawler/service_tree_crawler.go
   func (c *ServiceTreeCrawler) CrawlAllServiceTrees(ctx context.Context)
   ```

2. **使用条件断点**：
   - 右键断点 → "编辑断点"
   - 添加条件：`len(views.Views) > 0`

3. **监视表达式**：
   ```
   len(views.Views)
   ctx.Err()
   treeData
   ```

### 场景3：调试配置加载

1. **设置断点**：
   ```go
   // cmd/root.go
   func GetConfig() *Config
   ```

2. **查看配置结构**：
   - 在变量面板展开 `config` 对象
   - 监视 `config.CMDB.Auth.APIKey`

## 🚨 常见问题

### 1. 断点不生效

**原因**：可能没有使用调试模式编译
**解决**：确保 `buildFlags` 包含 `-gcflags='all=-N -l'`

### 2. 连接远程调试失败

**检查**：
```bash
# 确保调试服务器正在运行
lsof -i :2345

# 检查防火墙设置
```

### 3. 变量显示不完整

**解决**：
- 在设置中搜索 "go debug"
- 调整 `go.delveConfig` 相关设置

### 4. 调试过程中程序崩溃

**解决**：
- 检查panic和错误日志
- 使用 `recover` 捕获panic
- 在崩溃点前设置断点

## 🎨 Cursor特色功能

### 1. AI辅助调试
- 使用Cursor的AI功能分析调试信息
- 询问AI："这个变量的值为什么是nil？"
- 让AI解释复杂的调用堆栈

### 2. 智能断点建议
- Cursor可以建议在哪里设置断点
- 基于代码分析提供调试策略

### 3. 自动修复建议
- 当发现问题时，Cursor可能提供修复建议
- 结合调试信息和AI分析

## 📝 调试最佳实践

1. **从主函数开始**：首先在 `main()` 设置断点
2. **分层调试**：按模块逐个调试
3. **使用条件断点**：避免在循环中频繁停止
4. **监视关键变量**：添加重要变量到监视面板
5. **利用调用堆栈**：理解函数调用关系
6. **组合使用日志**：调试和日志相结合

## 🔗 相关链接

- [Go调试扩展文档](https://code.visualstudio.com/docs/languages/go#_debugging)
- [Delve调试器](https://github.com/go-delve/delve)
- [Cursor官方文档](https://cursor.sh/docs)

---

**Happy Debugging with Cursor!** 🎯✨ 