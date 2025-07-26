import hashlib
import requests
from urllib.parse import urlparse
import warnings

# 抑制urllib3的OpenSSL警告
warnings.filterwarnings('ignore', message='urllib3 v2 only supports OpenSSL 1.1.1+')

BASE_URL = "https://cmdb.veops.cn"
KEY = "d0a8fb5aeedf466c92cc5142a18d1a68"
SECRET = "DSGYH81jqfw~%A&vgyJKXrO*UFVaW2xt"


def build_api_key(path, params):
    values = "".join([str(params[k]) for k in sorted((params or {}).keys())
                      if k not in ("_key", "_secret") and not isinstance(params[k], (dict, list))])
    _secret = "".join([path, SECRET, values]).encode("utf-8")
    signature = hashlib.sha1(_secret).hexdigest()
    params["_secret"] = signature
    params["_key"] = KEY
    
    # 调试信息
    print(f"请求路径: {path}")
    print(f"参数: {params}")
    print(f"签名字符串: {path + SECRET + values}")
    print(f"签名结果: {signature}")
    print()

    return params


def api_request(endpoint, payload):
    url = f"{BASE_URL}{endpoint}"
    payload = build_api_key(urlparse(url).path, payload)
    
    response = requests.get(url, params=payload)
    print(f"HTTP状态码: {response.status_code}")
    print(f"完整URL: {response.url}")
    
    return response.json()


# 示例调用
if __name__ == "__main__":
    # 测试多种查询方式
    test_cases = [
        {
            "name": "查询所有CI（无筛选）",
            "endpoint": "/api/v0.1/ci/s",
            "payload": {
                "count": 10
            }
        },
        {
            "name": "查询所有CI（不使用use_id_filter）", 
            "endpoint": "/api/v0.1/ci/s",
            "payload": {
                "q": "_type:(39)",
                "count": 10
            }
        },
        {
            "name": "测试服务树视图列表",
            "endpoint": "/api/v0.1/preference/relation/view",
            "payload": {}
        },
        {
            "name": "查询CI类型定义",
            "endpoint": "/api/v0.1/ci_types",
            "payload": {}
        },
        {
            "name": "产品线 CI类型39 (简化查询)",
            "endpoint": "/api/v0.1/ci/s",
            "payload": {
                "q": "_type:39",  # 不使用括号
                "count": 10
            }
        }
    ]
    
    for test_case in test_cases:
        print(f"=== 测试: {test_case['name']} ===")
        try:
            result = api_request(test_case['endpoint'], test_case['payload'])
            if 'total' in result:
                print(f"总数: {result['total']}")
                print(f"当前页数据数量: {result.get('numfound', 0)}")
                if result.get('result'):
                    print(f"第一条数据: {result['result'][0]}")
            elif isinstance(result, list):
                print(f"列表长度: {len(result)}")
                if result:
                    print(f"第一条数据: {result[0]}")
            else:
                print(f"返回结果: {result}")
            print("-" * 50)
        except Exception as e:
            print(f"API调用失败: {e}")
            print("-" * 50)
