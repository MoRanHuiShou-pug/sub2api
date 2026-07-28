# NewAPI — 分组与Token管理接口

Base URL：`https://{your-newapi-instance}`

---

## 分组接口

### GET /api/group/

获取所有分组列表。权限：`AdminAuth`

**响应**
```json
{
  "success": true,
  "data": [
    "default",
    "vip",
    "free"
  ]
}
```

> NewAPI 的分组是简单的字符串标签，无独立的分组对象（无 rate_multiplier 等字段）。分组通过 Channel 的 `group` 字段（逗号分隔）来关联渠道。

### GET /api/user/groups

获取可用分组列表（无需认证，公开接口）。

### GET /api/user/self/groups

获取当前用户可用的分组。权限：`UserAuth`

---

## Token / API Key 接口

Token 是用户用于调用中转 API 的密钥（对应 Sub2API 的 API Key 概念）。

### GET /api/token/

获取当前用户的 Token 列表（key 字段被掩码：前4+****+后4）。权限：`UserAuth`

**响应**
```json
{
  "success": true,
  "data": {
    "tokens": [
      {
        "id":                  1,
        "name":                "我的密钥",
        "key":                 "sk-1234****abcd",
        "status":              1,
        "created_time":        1749600000,
        "accessed_time":       1753700000,
        "expired_time":        -1,
        "remain_quota":        0,
        "unlimited_quota":     true,
        "model_limits_enabled": false,
        "model_limits":        "",
        "used_quota":          12345,
        "group":               "default",
        "cross_group_retry":   false
      }
    ]
  }
}
```

**Token status 值**

| status | 含义 |
|---|---|
| 1 | 启用 |
| 2 | 禁用 |
| 3 | 已过期 |
| 4 | 已耗尽 |

### GET /api/token/:id

获取单个 Token 详情。

### POST /api/token/

创建新 Token。权限：`UserAuth`

**请求体**
```json
{
  "name":                 "密钥名称",
  "expired_time":         -1,
  "remain_quota":         0,
  "unlimited_quota":      true,
  "model_limits_enabled": false,
  "model_limits":         "",
  "group":                "default",
  "allow_ips":            null
}
```

### PUT /api/token/

更新 Token（id 在请求体中）。权限：`UserAuth`

### DELETE /api/token/:id

删除单个 Token。

### POST /api/token/batch

批量删除 Token。

**请求体**：`{"ids": [1, 2, 3]}`

### POST /api/token/:id/key

获取 Token 完整 key（明文），有频率限制。

**响应**
```json
{
  "success": true,
  "data":    "sk-full-key-string"
}
```

### POST /api/token/batch/keys

批量获取完整 key（最多 100 条）。

**请求体**：`{"ids": [1, 2, 3]}`

---

## Token 数据结构

```go
type Token struct {
  Id                 int     `json:"id"`
  UserId             int     `json:"user_id"`
  Key                string  `json:"key"`                 // 列表中被掩码
  Status             int     `json:"status"`
  Name               string  `json:"name"`
  CreatedTime        int64   `json:"created_time"`
  AccessedTime       int64   `json:"accessed_time"`
  ExpiredTime        int64   `json:"expired_time"`        // -1 = 永不过期
  RemainQuota        int     `json:"remain_quota"`
  UnlimitedQuota     bool    `json:"unlimited_quota"`
  ModelLimitsEnabled bool    `json:"model_limits_enabled"`
  ModelLimits        string  `json:"model_limits"`        // 逗号分隔
  AllowIps           *string `json:"allow_ips"`           // 换行分隔的 CIDR/IP
  UsedQuota          int     `json:"used_quota"`
  Group              string  `json:"group"`
  CrossGroupRetry    bool    `json:"cross_group_retry"`
}
```
