# NewAPI — 渠道管理接口

Base URL：`https://{your-newapi-instance}`

认证：`Authorization: Bearer <access_token>`（需要 `AdminAuth` 权限）

---

## GET /api/channel/

获取渠道列表。

> 注意：列表接口中 `key` 字段被省略（安全保护）。如需获取 key，使用 `/api/channel/:id/key`。

**查询参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` | int | 页码（默认 1） |
| `page_size` | int | 每页数量（默认 10） |

**响应**
```json
{
  "success": true,
  "data": {
    "channels": [
      {
        "id":               1,
        "name":             "OpenAI 官方",
        "type":             1,
        "status":           1,
        "weight":           100,
        "base_url":         null,
        "models":           "gpt-4o,gpt-4o-mini",
        "group":            "default,vip",
        "used_quota":       12345,
        "balance":          10.5,
        "balance_updated_time": 1753700000,
        "response_time":    523,
        "test_time":        1753700000,
        "priority":         0,
        "auto_ban":         1,
        "model_mapping":    "{\"gpt-4\":\"gpt-4o\"}",
        "remark":           null
      }
    ],
    "total": 10
  }
}
```

**渠道 type 值（部分常用）**

| type | 对应平台 |
|---|---|
| 1 | OpenAI |
| 3 | Azure OpenAI |
| 14 | Anthropic Claude |
| 24 | Gemini |
| 25 | Mistral |
| 33 | DeepSeek |
| 36 | xAI Grok |

**渠道 status 值**

| status | 含义 |
|---|---|
| 1 | 启用 |
| 2 | 手动禁用 |
| 3 | 自动禁用（测试失败触发） |

---

## GET /api/channel/search

搜索渠道。

**查询参数**：`keyword=<string>`

---

## GET /api/channel/:id

获取单个渠道详情（key 字段仍被省略）。

---

## POST /api/channel/

创建渠道。权限：`ChannelSensitiveWrite`

**请求体**
```json
{
  "name":           "渠道名称",
  "type":           1,
  "key":            "sk-xxx",
  "base_url":       "https://api.example.com",
  "models":         "gpt-4o,gpt-4o-mini",
  "group":          "default",
  "weight":         100,
  "priority":       0,
  "auto_ban":       1,
  "model_mapping":  "{\"gpt-4\":\"gpt-4o\"}",
  "remark":         "备注"
}
```

---

## PUT /api/channel/

更新渠道（注意：是 PUT 到 `/api/channel/` 而非 `/:id`，id 在请求体中传递）。权限：`ChannelWrite`

**请求体**：同创建，额外包含 `"id": 1`。

---

## DELETE /api/channel/:id

删除渠道。权限：`ChannelSensitiveWrite`

---

## POST /api/channel/batch

批量删除渠道。

**请求体**
```json
{
  "ids": [1, 2, 3]
}
```

---

## GET /api/channel/test/:id

测试单个渠道连通性。权限：`ChannelOperate`

**响应**
```json
{
  "success": true,
  "message": "OK",
  "time":    0.523
}
```

---

## GET /api/channel/test

测试所有渠道（异步批量）。权限：`ChannelOperate`

---

## GET /api/channel/update_balance/:id

刷新渠道的上游余额。权限：`ChannelOperate`

**响应**
```json
{
  "success": true,
  "data": {
    "balance": 10.5
  }
}
```

---

## POST /api/channel/:id/status

修改渠道启用状态。权限：`ChannelOperate`

**请求体**
```json
{
  "status": 1
}
```

---

## POST /api/channel/status/batch

批量修改渠道状态。

**请求体**
```json
{
  "ids":    [1, 2, 3],
  "status": 1
}
```

---

## GET /api/channel/:id/key

获取渠道完整 key（明文）。

**权限**：`RootAuth + SecureVerification`（高敏感操作，需 Root 权限 + 二次验证）

**响应**
```json
{
  "success": true,
  "data":    "sk-xxx-full-key"
}
```

---

## POST /api/channel/copy/:id

复制渠道（克隆配置）。权限：`ChannelSensitiveWrite`

---

## GET /api/channel/models

获取渠道已配置的模型列表汇总。权限：`ChannelRead`

---

## 渠道数据结构

```go
type Channel struct {
  Id                 int      `json:"id"`
  Type               int      `json:"type"`
  Key                string   `json:"key"`               // 列表中被省略
  Status             int      `json:"status"`
  Name               string   `json:"name"`
  Weight             *uint    `json:"weight"`             // 调度权重
  BaseURL            *string  `json:"base_url"`
  Models             string   `json:"models"`             // 逗号分隔
  Group              string   `json:"group"`              // 逗号分隔分组名
  UsedQuota          int64    `json:"used_quota"`
  Balance            float64  `json:"balance"`            // 上游余额 (USD)
  BalanceUpdatedTime int64    `json:"balance_updated_time"`
  ResponseTime       int      `json:"response_time"`      // 最近测试延迟(ms)
  TestTime           int64    `json:"test_time"`
  ModelMapping       *string  `json:"model_mapping"`      // JSON 映射
  Priority           *int64   `json:"priority"`
  AutoBan            *int     `json:"auto_ban"`           // 1=自动封禁
  Remark             *string  `json:"remark"`
  ParamOverride      *string  `json:"param_override"`     // 请求参数覆盖(JSON)
  HeaderOverride     *string  `json:"header_override"`    // 请求 Header 覆盖(JSON)
}
```
