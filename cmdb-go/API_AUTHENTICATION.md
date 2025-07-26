# CMDB API 认证配置指南

## 🎯 重要发现

通过分析[Veops CMDB API官方文档](https://veops.cn/docs/docs/cmdb/cmdb_api)，我们发现CMDB系统使用**API Key/Secret签名认证**方式，而不是JWT Token认证。

## 📋 认证方式说明

| 认证方式 | 用途 | 获取方式 |
|---------|------|----------|
| **API Key/Secret** ✅ | API调用（推荐） | ACL系统中获取 |
| **JWT Token** | Web前端界面 | 登录接口获取 |
| **用户名密码** | 备用方式 | 直接配置 |

## 🔑 获取API凭据

### 1. 登录CMDB系统
访问您的CMDB系统（如：`https://cmdb.veops.cn`）

### 2. 进入ACL系统
在系统中找到**ACL系统**或**权限管理**模块

### 3. 查看API密钥
根据官方文档说明：
> 每个用户会自动生成一个 `api key` 和一个 `secret`，在ACL系统里可查看到

找到您的：
- **API Key**: 类似 `abcd1234567890`
- **API Secret**: 类似 `xyz9876543210`

## ⚙️ 配置API凭据

### 方法1：修改配置文件
编辑 `config/config.yaml`：

```yaml
cmdb:
  base_url: "https://cmdb.veops.cn"
  api_version: "api/v0.1"
  login_api_version: "api/v1"
  auth:
    # 将YOUR_API_KEY和YOUR_API_SECRET替换为真实值
    api_key: "your_real_api_key_here"
    api_secret: "your_real_api_secret_here"
    # 备用认证
    username: "your_username"
    password: "your_password"
```

### 方法2：环境变量（推荐）
```bash
export CMDB_AUTH_API_KEY="your_real_api_key_here"
export CMDB_AUTH_API_SECRET="your_real_api_secret_here"
export CMDB_BASE_URL="https://your-cmdb-server.com"
```

### 方法3：命令行参数
```bash
./cmdb-crawler crawl \
  --config ./config/config.yaml \
  --verbose
```

## 🚀 验证配置

运行爬取命令：
```bash
./cmdb-crawler crawl --verbose
```

**正确配置的输出示例**：
```
{"level":"info","msg":"Using API Key authentication","api_key":"abcd1234567890"}
{"level":"info","msg":"Successfully fetched relation views","view_count":3}
```

**错误配置的输出示例**：
```
{"level":"error","msg":"API returned status 401: unauthorized"}
```

## 🔐 API签名算法

我们的实现严格按照官方文档的签名算法：

1. **收集参数**：除`_key`和`_secret`外的所有参数
2. **参数排序**：按参数名字典序排序
3. **拼接字符串**：`url_path` + `secret` + `参数值`
4. **计算签名**：`SHA1(拼接字符串)`的十六进制值

**实现代码**：
```go
func (c *CMDBClient) buildSignature(urlPath string, params map[string]string) string {
    // 1. 收集并排序参数名
    var keys []string
    for k := range params {
        if k != "_key" && k != "_secret" {
            keys = append(keys, k)
        }
    }
    sort.Strings(keys)
    
    // 2. 拼接参数值
    var values []string
    for _, k := range keys {
        values = append(values, params[k])
    }
    paramValues := strings.Join(values, "")
    
    // 3. 构建签名字符串
    signStr := urlPath + c.apiSecret + paramValues
    
    // 4. 计算SHA1
    h := sha1.New()
    h.Write([]byte(signStr))
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

## 🎯 测试连接

使用正确的API凭据后，您应该能够：

1. **获取服务树视图**：`/api/v0.1/preference/relation/view`
2. **搜索CI实例**：`/api/v0.1/ci/s`
3. **查询CI关系**：`/api/v0.1/ci_relations/s`

## 📞 技术支持

如果遇到问题：

1. **检查API凭据**：确保从ACL系统获取的Key和Secret正确
2. **验证权限**：确保用户有访问CMDB API的权限
3. **网络连接**：确保能访问CMDB服务器
4. **日志调试**：使用`--verbose`参数查看详细日志

## 🌟 成功案例

一旦配置正确，您将看到类似输出：
```
✅ 成功连接到CMDB系统
✅ API Key认证通过
✅ 获取到 X 个服务树视图
✅ 开始爬取服务树数据...
✅ 数据已导出到：./output/service_tree_data.json
```

---

**参考文档**：[Veops CMDB API文档](https://veops.cn/docs/docs/cmdb/cmdb_api) 