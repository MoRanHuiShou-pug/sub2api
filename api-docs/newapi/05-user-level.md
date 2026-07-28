# NewAPI — 用户态可用接口

> 本文档仅覆盖**普通用户登录后可访问**的接口（无需 admin/root 角色）。
> 认证接口（登录/刷新/self）见 [01-auth.md](./01-auth.md)。

Base URL：`https://{your-newapi-instance}`

认证：Cookie-based session（HttpOnly），**同时**需要 `New-Api-User: <user_id>` header。

> ⚠️ **部署差异**：标准 new-api 仅需 session cookie。部分定制部署（如 `new-api-customized` 系列）额外要求 `New-Api-User: <用户ID>` header，否则所有接口均返回 `401 Unauthorized`。用户 ID 可从登录响应的 `data.id` 获取。

---

## 分组相关

### GET /api/user/groups

**获取可用分组列表及倍率**。**无需认证**（公开接口），但登录后可获取个性化倍率。

> 这是 NewAPI 用户态获取分组倍率的**核心接口**。

**响应**（登录后调用，返回该用户可用的分组）
```json
{
  "success": true,
  "data": {
    "default": {
      "ratio": 1.0,
      "desc":  "默认分组"
    },
    "vip": {
      "ratio": 0.8,
      "desc":  "VIP 优惠分组"
    },
    "auto": {
      "ratio": "自动",
      "desc":  "自动选择最优分组"
    }
  }
}
```

`ratio` 字段为该用户在该分组的倍率（float，"自动" 为特殊值）。

### GET /api/user/self/groups

获取当前登录用户所属分组（需认证）。

**响应**
```json
{
  "success": true,
  "data": {
    "default": {
      "ratio": 1.0,
      "desc":  "默认分组"
    }
  }
}
```

---

## 定价与倍率

### GET /api/pricing

**获取模型定价和分组倍率**。登录后调用可获取个性化数据（根据用户所属分组过滤）。

> 这是 NewAPI 获取 `group_ratio`（分组计费倍率）最完整的用户态接口。

**响应**
```json
{
  "success": true,
  "data": {
    "gpt-4o": {
      "input":   0.005,
      "output":  0.015,
      "type":    "tokens"
    }
  },
  "group_ratio": {
    "default": 1.0,
    "vip":     0.8
  },
  "usable_group": {
    "default": "默认分组",
    "vip":     "VIP 优惠分组"
  },
  "supported_endpoint": {
    "chat":       true,
    "embeddings": true,
    "images":     false
  },
  "auto_groups": ["vip"],
  "vendors": []
}
```

**字段说明**

| 字段 | 说明 |
|---|---|
| `group_ratio` | 各分组的计费倍率（可能受用户组别影响） |
| `usable_group` | 当前用户可用的分组及其描述 |
| `data` | 各模型的 input/output 单价（USD/1K tokens） |

---

## 系统状态

### GET /api/status

系统配置状态（无需认证，公开接口）。

> ⚠️ 此接口返回系统**配置信息**（OAuth 是否开启、系统名称等），**不包含账号健康状态**（无 error_accounts 等字段）。

**响应（部分字段）**
```json
{
  "success": true,
  "data": {
    "version":                "1.x.x",
    "start_time":             1753700000,
    "system_name":            "New API",
    "register_enabled":       true,
    "password_login_enabled": true,
    "github_oauth":           true,
    "email_verification":     false,
    "quota_per_unit":         500000,
    "display_in_currency":    true
  }
}
```

---

## 用户信息与模型

### GET /api/user/self

获取当前用户完整信息（需认证）。见 [01-auth.md](./01-auth.md)。

### GET /api/user/models

获取当前用户可用的模型列表（需认证）。

**响应**
```json
{
  "success": true,
  "data": ["gpt-4o", "gpt-4o-mini", "claude-opus-4-5"]
}
```

---

## 使用记录

### GET /api/log/self

获取当前用户的请求日志（需认证）。

**查询参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `p` | int | 页码 |
| `page_size` | int | 每页数量 |
| `type` | int | 日志类型筛选 |
| `start_timestamp` | int | 开始时间戳 |
| `end_timestamp` | int | 结束时间戳 |

### GET /api/log/self/stat

当前用户的请求统计汇总（需认证）。

**响应**
```json
{
  "success": true,
  "data": {
    "quota":          12345,
    "token_used":     12345,
    "count":          100,
    "prompt_tokens":  800000,
    "completion_tokens": 50000
  }
}
```

### GET /api/data/self

当前用户的配额使用时序数据（需认证）。

### GET /api/data/flow/self

当前用户的流量时序数据（需认证）。

### GET /api/usage/token

通过 Token Key 查询使用量（无需用户登录，使用 Token 本身认证）。

---

## Token / API Key 管理

Token 是用户用于调用 NewAPI 中转网关的密钥（对应 Sub2API 的 API Key 概念）。

### GET /api/token/

获取当前用户的 Token 列表（需认证）。

> ⚠️ **列表中 `key` 字段被掩码**（前4+****+后4），如需明文 key 需额外调用 `/api/token/:id/key`。

**响应**
```json
{
  "success": true,
  "data": {
    "tokens": [
      {
        "id":                   1,
        "name":                 "我的密钥",
        "key":                  "sk-1234****abcd",
        "status":               1,
        "created_time":         1749600000,
        "accessed_time":        1753700000,
        "expired_time":         -1,
        "remain_quota":         0,
        "unlimited_quota":      true,
        "model_limits_enabled": false,
        "model_limits":         "",
        "used_quota":           12345,
        "group":                "default",
        "cross_group_retry":    false
      }
    ]
  }
}
```

**status 值**：`1` = 启用，`2` = 禁用，`3` = 已过期，`4` = 已耗尽

---

### GET /api/token/:id

获取单个 Token 详情（key 同样被掩码）。

---

### POST /api/token/

创建新 Token。

> ⚠️ **创建响应不包含 key 明文**，仅返回 `{"success": true, "message": ""}` 和 Token ID。
> 创建后需调用 `POST /api/token/:id/key` 才能获取完整 key，请立即保存。

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

**字段说明**

| 字段 | 必填 | 说明 |
|---|---|---|
| `name` | ✅ | 密钥名称 |
| `group` | ❌ | 绑定分组名（默认 `default`） |
| `expired_time` | ❌ | Unix 时间戳，`-1` = 永不过期 |
| `remain_quota` | ❌ | 配额上限（配合 `unlimited_quota`），单位为 NewAPI 内部 quota 单位 |
| `unlimited_quota` | ❌ | `true` = 无限额度 |
| `model_limits_enabled` | ❌ | 是否开启模型白名单 |
| `model_limits` | ❌ | 允许的模型列表（逗号分隔） |
| `allow_ips` | ❌ | IP 白名单（换行分隔的 CIDR/IP，`null` = 不限） |

**响应**
```json
{
  "success": true,
  "message": ""
}
```

---

### POST /api/token/:id/key

**获取 Token 完整 key（明文）**，有频率限制。

**响应**
```json
{
  "success": true,
  "data":    "sk-full-key-string"
}
```

---

### POST /api/token/batch/keys

批量获取完整 key（最多 100 条）。

**请求体**：`{"ids": [1, 2, 3]}`

**响应**
```json
{
  "success": true,
  "data": {
    "1": "sk-full-key-1",
    "2": "sk-full-key-2"
  }
}
```

---

### PUT /api/token/

更新 Token（id 在请求体中传递）。

**请求体**：同创建字段，额外包含 `"id": 1`。

---

### DELETE /api/token/:id

删除单个 Token（不可恢复）。

### POST /api/token/batch

批量删除 Token。**请求体**：`{"ids": [1, 2, 3]}`

---

## 与 Sub2API 用户态对比

| 能力 | Sub2API 用户态 | NewAPI 用户态 |
|---|---|---|
| 获取分组倍率 | ✅ `GET /api/v1/groups/rates`<br>返回 `rate_multiplier`（float） | ✅ `GET /api/user/groups`<br>返回 `ratio`（float） |
| 获取可用分组 | ✅ `GET /api/v1/groups/available` | ✅ `GET /api/user/groups` |
| 获取模型定价 | ❌ 无专用接口 | ✅ `GET /api/pricing` |
| 使用统计 | ✅ `/usage/dashboard/stats` | ✅ `GET /api/log/self/stat` |
| 创建 API Key | ✅ `POST /keys`（响应含完整 key） | ✅ `POST /api/token/`（⚠️ 需再调用 `/:id/key`） |
| 列出 API Key | ✅ `GET /keys`（key 明文） | ✅ `GET /api/token/`（key 掩码） |
| 删除 API Key | ✅ `DELETE /keys/:id` | ✅ `DELETE /api/token/:id` |
| 账号健康状态 | ❌ 仅管理员 | ❌ 仅管理员 |
| 系统健康状态 | ❌ 仅管理员 | ⚠️ `GET /api/status` 仅有配置信息，无账号健康 |
