# Sub2API — 账号管理接口

Base URL：`https://{your-sub2api-instance}/api/v1`

认证：`Authorization: Bearer <access_token>`（需要 admin 角色）

---

## GET /admin/accounts

获取账号列表。

**查询参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` | int | 页码（默认 1） |
| `page_size` | int | 每页数量（默认 20） |
| `platform` | string | 按平台筛选（anthropic/openai/gemini/antigravity/grok） |
| `type` | string | 按认证类型筛选（oauth/apikey/upstream 等） |
| `status` | string | 按状态筛选（active/error/disabled） |

**响应**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id":              4,
        "name":            "Agnes - 官方api",
        "notes":           "admin@example.com",
        "platform":        "openai",
        "type":            "apikey",
        "status":          "active",
        "error_message":   null,
        "priority":        1,
        "concurrency":     10,
        "rate_multiplier": 1.0,
        "schedulable":     true,
        "rate_limited_at": null,
        "last_used_at":    "2026-06-10T23:51:34+08:00",
        "created_at":      "2026-06-10T18:43:14+08:00"
      }
    ],
    "total": 3,
    "page":  1,
    "page_size": 20
  }
}
```

**账号状态说明**

| status | 含义 |
|---|---|
| `active` | 正常，可调度 |
| `error` | 出错（Token 失效、网络不通等） |
| `disabled` | 手动禁用 |
| `rate_limited` | 触发速率限制（429），临时不可用 |
| `overload` | 上游过载（529），临时不可用 |

**平台常量**

| platform | 对应服务 |
|---|---|
| `anthropic` | Anthropic Claude 直连 |
| `openai` | OpenAI 直连 |
| `gemini` | Google Gemini |
| `antigravity` | Antigravity（Claude 兼容中转） |
| `grok` | xAI Grok |

**认证类型常量**

| type | 含义 |
|---|---|
| `oauth` | OAuth 2.0 令牌（full scope） |
| `setup-token` | Setup Token（inference only） |
| `apikey` | API Key（可含自定义 base_url） |
| `upstream` | 上游透传（base_url + api_key） |
| `bedrock` | AWS Bedrock（SigV4 或 API Key） |
| `service_account` | Google Service Account（Vertex AI） |

---

## GET /admin/accounts/:id

获取单个账号详情（含 credentials 脱敏字段）。

---

## POST /admin/accounts

创建新账号。

**请求体**
```json
{
  "name":            "渠道名称",
  "platform":        "openai",
  "type":            "apikey",
  "credentials": {
    "base_url": "https://api.example.com",
    "api_key":  "sk-xxx"
  },
  "concurrency":     10,
  "priority":        1,
  "rate_multiplier": 1.0,
  "status":          "active",
  "notes":           "备注（可选）"
}
```

**响应**：返回创建后的账号对象（含 `id`）。

---

## PUT /admin/accounts/:id

更新账号，字段与 POST 相同，全量覆盖。

---

## DELETE /admin/accounts/:id

删除账号（软删除）。

---

## POST /admin/accounts/:id/test

测试账号连通性，返回是否可用及延迟。

**响应**
```json
{
  "code": 0,
  "data": {
    "success":      true,
    "latency_ms":   523,
    "error":        null
  }
}
```

---

## POST /admin/accounts/:id/refresh

触发账号 Token 刷新（适用于 OAuth/setup-token 类型）。

---

## POST /admin/accounts/:id/clear-error

清除账号的错误状态，重置为 active。

---

## POST /admin/accounts/:id/schedulable

设置账号是否参与调度。

**请求体**
```json
{ "schedulable": true }
```

---

## POST /admin/accounts/:id/clear-rate-limit

清除账号的速率限制状态。

---

## GET /admin/accounts/:id/stats

获取账号使用统计（请求数、Token 数、费用）。

---

## GET /admin/accounts/:id/today-stats

获取账号今日统计数据。

---

## 批量操作

| 路径 | 功能 |
|---|---|
| `POST /admin/accounts/batch-refresh` | 批量刷新 Token，body: `{"account_ids":[1,2,3]}` |
| `POST /admin/accounts/batch-clear-error` | 批量清除错误状态 |
| `POST /admin/accounts/bulk-update` | 批量更新字段（priority/rate_multiplier 等） |
| `POST /admin/accounts/batch` | 批量创建账号 |

---

## GET /admin/accounts/data

导出账号数据（需 2FA Step-Up 验证）。

## POST /admin/accounts/data

导入账号数据。
