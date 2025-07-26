# CMDB Crawler 调试指南

本文档介绍如何使用dlv调试器来调试CMDB服务树数据爬取工具。

## 🔧 准备工作

### 1. 安装dlv调试器

```bash
make install-dlv
```

或者手动安装：
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### 2. 验证安装

```bash
dlv version
```

## 🚀 调试方法

### 1. 直接调试模式（推荐新手）

最简单的调试方式，直接启动交互式调试：

```bash
make debug-direct
```

这会启动一个交互式的调试会话，你可以：
- 设置断点：`break main.main`
- 继续执行：`continue` 或 `c`
- 单步执行：`next` 或 `n`
- 步入函数：`step` 或 `s`
- 查看变量：`print variable_name`

### 2. Headless模式调试（推荐IDE用户）

启动调试服务器，可以用IDE连接：

```bash
make debug
```

默认监听 `localhost:2345`，你可以：
- 使用GoLand/VS Code等IDE连接到这个端口
- 或者用命令行连接：`dlv connect localhost:2345`

### 3. 调试特定命令

调试特定的命令参数：

```bash
make debug-cmd CMD="crawl --verbose --format yaml"
```

### 4. 自定义调试端口

使用自定义端口进行调试：

```bash
make debug DEBUG_PORT=3456
```

### 5. 调试测试

调试单元测试：

```bash
make debug-test
```

### 6. 附加到运行中的进程

如果程序已经在运行，可以附加到进程：

```bash
# 先找到进程ID
ps aux | grep cmdb-crawler

# 然后附加调试
make debug-attach
# 输入进程ID
```

## 🎯 常用调试命令

### 断点操作
```bash
# 在函数入口设置断点
break main.main
break internal/client.(*CMDBClient).GetRelationViews

# 在文件的某一行设置断点
break main.go:25

# 查看所有断点
breakpoints

# 删除断点
clear 1  # 删除断点ID为1的断点
```

### 程序控制
```bash
# 继续执行
continue
c

# 单步执行（不进入函数内部）
next
n

# 单步执行（进入函数内部）
step 
s

# 跳出当前函数
stepout
so

# 重启程序
restart
r
```

### 变量查看
```bash
# 查看变量值
print variable_name
p variable_name

# 查看变量类型
whatis variable_name

# 查看局部变量
locals

# 查看函数参数
args

# 查看调用栈
stack
bt
```

### 协程调试（Go特色）
```bash
# 查看所有协程
goroutines

# 切换到特定协程
goroutine 1

# 查看当前协程信息
goroutine
```

## 📋 调试场景示例

### 场景1：调试API认证问题

1. 在认证相关函数设置断点：
```bash
break internal/client.(*CMDBClient).SetAPICredentials
break internal/client.(*CMDBClient).buildSignature
```

2. 运行程序并查看认证过程：
```bash
make debug-direct
(dlv) break internal/client.(*CMDBClient).buildSignature
(dlv) continue
(dlv) print urlPath
(dlv) print c.apiSecret
(dlv) print params
```

### 场景2：调试数据爬取逻辑

1. 在爬取函数设置断点：
```bash
break internal/crawler.(*ServiceTreeCrawler).CrawlAllServiceTrees
break internal/crawler.(*ServiceTreeCrawler).crawlServiceTree
```

2. 逐步跟踪爬取过程：
```bash
(dlv) continue
(dlv) print treeData
(dlv) next
(dlv) print response
```

### 场景3：调试配置加载

1. 在配置相关函数设置断点：
```bash
break cmd.GetConfig
break cmd.mergeFlags
```

2. 查看配置值：
```bash
(dlv) continue
(dlv) print config.CMDB.Auth.APIKey
(dlv) print config.CMDB.BaseURL
```

## 🔍 高级调试技巧

### 1. 条件断点

只有满足特定条件时才触发的断点：

```bash
break main.go:50
condition 1 variable_name == "特定值"
```

### 2. 监视点（Watchpoint）

监视变量值的变化：

```bash
# 当变量值改变时暂停
watch variable_name
```

### 3. 反汇编调试

查看汇编代码（高级用户）：

```bash
disassemble
```

### 4. 内存查看

查看内存内容：

```bash
examine 0x地址
```

## 🛠️ IDE集成

### Cursor/VS Code配置

我们已经为Cursor提供了完整的调试配置！查看 **[Cursor调试指南](CURSOR_DEBUG_GUIDE.md)** 获取详细说明。

项目中的 `.vscode/launch.json` 包含了7种不同的调试配置：
- 直接启动调试
- 连接到远程调试服务器
- 自定义参数调试
- 测试调试等

**快速开始**：
1. 在Cursor中按 `Cmd+Shift+D` 打开调试面板
2. 选择 "Debug CMDB Crawler" 配置
3. 按 `F5` 开始调试

### GoLand配置

1. Run/Debug Configurations → Add New → Go Remote
2. Host: `localhost`
3. Port: `2345`
4. 先运行 `make debug`，然后在GoLand中启动Remote Debug

## 🚨 常见问题

### 1. 找不到dlv命令

确保 `$GOPATH/bin` 在你的 `$PATH` 中：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 2. 调试时程序无响应

检查是否有其他调试会话占用端口：

```bash
lsof -i :2345
kill -9 <PID>
```

### 3. 无法设置断点

确保使用调试版本构建（`make build-debug`），它禁用了优化。

### 4. 协程调试困难

使用 `goroutines` 命令查看所有协程，然后用 `goroutine <id>` 切换。

## 📝 调试最佳实践

1. **先看日志再调试**：很多问题可以通过日志发现
2. **从入口开始**：从 `main` 函数开始设置断点
3. **分层调试**：按模块（client、crawler、output）分别调试
4. **使用条件断点**：避免在循环中频繁停止
5. **保存调试会话**：记录有用的断点和变量监视

## 🎓 学习资源

- [Delve官方文档](https://github.com/go-delve/delve/tree/master/Documentation)
- [Go调试技巧](https://golang.org/doc/gdb)
- [VS Code Go调试指南](https://code.visualstudio.com/docs/languages/go#_debugging)

---

**Happy Debugging!** 🐛✨ 