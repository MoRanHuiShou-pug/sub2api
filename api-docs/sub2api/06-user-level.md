# Sub2API — 用户态可用接口

> 本文档仅覆盖**普通用户登录后可访问**的接口（无需 admin 角色）。
> 认证接口（登录/刷新/me）见 [01-auth.md](./01-auth.md)。

Base URL：`https://{your-sub2api-instance}/api/v1`

认证：`Authorization: Bearer <access_token>`（普通用户 JWT 即可）

---

## 分组相关

### GET /groups/available

获取当前用户可用的分组列表。

**响应**
```json
{
  "code": 0,
  "data": [
    {
      "id":               3,
      "name":             "DeepSeek - 官",
      "platform":         "anthropic",
      "status":           "active",
      "rate_multiplier":  1.0,
      "description":      "opus = deepseek-v4-pro\nsonnet = deepseek-v4-flash",
      "subscription_type": "standard"
    }
  ]
}
```

---

### GET /groups/rates

**获取当前用户各分组的专属倍率配置**。上游管理同步分组倍率的核心接口。

> 若管理员为该用户的分组设置了专属倍率覆盖，此接口返回覆盖后的值；否则返回分组默认倍率。

**响应**
```json
{
  "code": 0,
  "data": {
    "groups": [
      {
        "group_id":        3,
        "group_name":      "DeepSeek - 官",
        "platform":        "anthropic",
        "rate_multiplier": 1.0
      },
      {
        "group_id":        4,
        "group_name":      "XiaoMi MiMo - 官",
        "platform":        "anthropic",
        "rate_multiplier": 0.8
      }
    ]
  }
}
```

`rate_multiplier` 含义：用户实际消费 = 原始 Token 成本 × 此倍率。1.0 = 原价，0.5 = 半价，2.0 = 双倍计费。

---

### GET /channels/available

获取当前用户可用的渠道列表（脱敏，不含 credentials）。

**响应**
```json
{
  "code": 0,
  "data": [
    {
      "id":       4,
      "name":     "Agnes - 官方api",
      "platform": "openai",
      "models":   ["gpt-5.5", "gpt-4o"]
    }
  ]
}
```

---

## 使用统计

### GET /usage/dashboard/stats

获取当前用户的使用总览统计。

**响应**
```json
{
  "code": 0,
  "data": {
    "total_requests":      309,
    "today_requests":      289,
    "total_tokens":        10134524,
    "today_tokens":        9493638,
    "total_input_tokens":  7188278,
    "today_input_tokens":  7112264,
    "total_output_tokens": 135254,
    "today_output_tokens": 122766,
    "total_cost":          41.11,
    "today_cost":          40.14,
    "average_duration_ms": 11170.5
  }
}
```

### GET /usage/dashboard/trend

当前用户的使用趋势（时序数据）。

**查询参数**：`start`、`end`（ISO 8601）、`granularity`（`hour`|`day`）

### GET /usage/dashboard/models

当前用户的模型使用分布。

### GET /usage/dashboard/snapshot-v2

综合快照（含趋势、模型分布等）。

### POST /usage/dashboard/api-keys-usage

指定 API Key 的使用详情。

**请求体**：`{"key_ids": [1, 2, 3]}`

---

## 个人账号

### GET /user/profile

获取当前用户完整 profile。

**响应**
```json
{
  "code": 0,
  "data": {
    "id":            1,
    "email":         "user@example.com",
    "username":      "用户名",
    "role":          "user",
    "balance":       100.0,
    "concurrency":   10,
    "status":        "active",
    "allowed_groups": [3, 4],
    "rpm_limit":     0
  }
}
```

### GET /user/platform-quotas

查询当前用户各平台的配额使用情况（如 Claude Pro 限额）。

---

## API Key 管理

路由前缀：`/api/v1/keys`

---

### GET /keys

列出当前用户的所有 API Key（分页）。

**查询参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` | int | 页码（默认 1） |
| `page_size` | int | 每页数量（默认 20） |
| `search` | string | 按名称搜索 |
| `status` | string | 按状态筛选（`active`/`inactive`） |
| `group_id` | int | 按分组筛选 |
| `sort_by` | string | 排序字段（默认 `created_at`） |
| `sort_order` | string | `asc`/`desc`（默认 `desc`） |

**响应**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id":           1,
        "user_id":      1,
        "key":          "sk-xxxxxxxxxxxxxxxxxxxx",
        "name":         "我的密钥",
        "group_id":     3,
        "status":       "active",
        "ip_whitelist": [],
        "ip_blacklist": [],
        "quota":        0,
        "quota_used":   12.5,
        "expires_at":   null,
        "last_used_at": "2026-07-28T10:00:00+08:00",
        "last_used_ip": "1.2.3.4",
        "created_at":   "2026-06-09T16:15:41+08:00",
        "updated_at":   "2026-07-28T10:00:00+08:00",
        "current_concurrency": 0,
        "rate_limit_5h":  0,
        "rate_limit_1d":  0,
        "rate_limit_7d":  0,
        "usage_5h":       0,
        "usage_1d":       0,
        "usage_7d":       0
      }
    ],
    "total":     5,
    "page":      1,
    "page_size": 20
  }
}
```

> `quota: 0` 表示无限额度；`key` 字段**明文返回**（Sub2API 列表接口不掩码）。

---

### GET /keys/:id

获取单个 API Key 详情，字段同上。

---

### POST /keys

创建新 API Key。**创建响应中直接包含完整 key 明文**，请立即保存。

**请求体**
```json
{
  "name":           "密钥名称",
  "group_id":       3,
  "custom_key":     null,
  "ip_whitelist":   [],
  "ip_blacklist":   [],
  "quota":          0,
  "expires_in_days": null,
  "rate_limit_5h":  0,
  "rate_limit_1d":  0,
  "rate_limit_7d":  0
}
```

**字段说明**

| 字段 | 必填 | 说明 |
|---|---|---|
| `name` | ✅ | 密钥名称 |
| `group_id` | ❌ | 绑定的分组 ID（null = 使用用户默认分组） |
| `custom_key` | ❌ | 自定义 key 后缀（null = 自动生成） |
| `quota` | ❌ | 配额上限 USD（0 = 不限） |
| `expires_in_days` | ❌ | 过期天数（null = 永不过期） |
| `rate_limit_5h/1d/7d` | ❌ | 滚动窗口限速 USD（0 = 不限） |

**响应**：同 GET /keys/:id 的 data 字段，包含完整 `key` 值。

---

### PUT /keys/:id

更新 API Key 配置。

**请求体**
```json
{
  "name":        "新名称",
  "group_id":    4,
  "status":      "active",
  "ip_whitelist": ["1.2.3.0/24"],
  "ip_blacklist": [],
  "quota":        100.0,
  "expires_at":  "2027-01-01T00:00:00Z",
  "reset_quota": false,
  "rate_limit_5h":          0,
  "rate_limit_1d":          0,
  "rate_limit_7d":          0,
  "reset_rate_limit_usage": false
}
```

> 所有字段均可选，仅传需要修改的字段。`ip_whitelist: []` 表示**清空**白名单；`ip_whitelist: null` 表示**不修改**。

---

### DELETE /keys/:id

删除 API Key（不可恢复）。

**响应**
```json
{
  "code": 0,
  "data": null
}
```

---

## 注意事项

| 能力 | 用户态是否可用 |
|---|---|
| 获取分组倍率（rate_multiplier） | ✅ `GET /groups/rates` |
| 获取可用分组列表 | ✅ `GET /groups/available` |
| 获取自己的使用统计 | ✅ `GET /usage/dashboard/stats` |
| 获取账号健康状态（error/normal 数量） | ❌ 仅管理员可用 |
| 获取系统级仪表盘健康统计 | ❌ 仅管理员可用 |
| 查看上游账号列表 | ❌ 仅管理员可用 |
