# NewAPI — 认证接口

Base URL：`https://{your-newapi-instance}`

认证方式：
- Dashboard 接口：`Authorization: Bearer <access_token>` 或 `Authorization: Bearer <pat>`
- 敏感操作附加：`X-Auth-Session: <session-id>`
- Relay 接口：`Authorization: Bearer sk-<key>` 或 `x-api-key: <key>`

> ⚠️ NewAPI 与 Sub2API 的认证机制有重要差异：
> - access_token 刷新通过 **HttpOnly Cookie** 静默完成，不在请求体传 refresh_token
> - 登录返回的 `session` 字段用于敏感操作的二次验证

---

## POST /api/user/login

邮箱密码登录。

**中间件**：`CriticalRateLimit + DisableCache + Turnstile`

**请求体**
```json
{
  "username": "admin@example.com",
  "password": "your-password"
}
```

> 注意：字段名是 `username`（不是 `email`），可填邮箱或用户名。

**响应 A — 登录成功**
```json
{
  "success": true,
  "message": "",
  "data": {
    "access_token":     "eyJhbGci...",
    "token_type":       "Bearer",
    "access_expires_at": 1753704000,
    "session":          "sess_xxx",
    "user": {
      "id":             1,
      "username":       "root",
      "display_name":   "管理员",
      "role":           100,
      "status":         1,
      "email":          "admin@example.com",
      "quota":          500000,
      "used_quota":     12345,
      "group":          "default",
      "admin_permissions": {}
    }
  }
}
```

**响应 B — 需要 2FA**
```json
{
  "success": true,
  "data": {
    "require_2fa":  true,
    "flow_token":   "flow_xxx",
    "expires_at":   1753704000
  }
}
```

**用户 role 值**

| role | 含义 |
|---|---|
| 1 | 普通用户 |
| 10 | 管理员 |
| 100 | Root（超级管理员） |

**用户 status 值**

| status | 含义 |
|---|---|
| 1 | 启用 |
| 2 | 禁用 |

---

## POST /api/user/login/2fa

完成 2FA 验证。

**请求体**
```json
{
  "flow_token": "flow_xxx",
  "code":       "123456"
}
```

**响应**：同 `/api/user/login` 的成功响应。

---

## POST /api/user/auth/refresh

刷新 access_token（使用 HttpOnly Cookie 中的 Refresh Token 静默续期）。

**请求**：无需请求体，Refresh Token 由浏览器自动携带 Cookie。

**后端实现说明（服务器端调用时）**：需携带登录时服务器设置的 Cookie（`refresh_token` cookie），直接以 POST 请求发送，服务器会返回新的 access_token 并更新 Cookie。

**响应**
```json
{
  "success": true,
  "data": {
    "access_token":     "eyJhbGci...",
    "token_type":       "Bearer",
    "access_expires_at": 1753704000
  }
}
```

> ⚠️ 与 Sub2API 的差异：NewAPI 的 refresh_token 存于 HttpOnly Cookie，服务端程序调用时需要持久化整个 Cookie Jar，而非单独的 token 字符串。

---

## POST /api/user/auth/logout

登出，清除 session 和 Cookie。

---

## GET /api/user/self

获取当前登录用户信息。

**请求头**：`Authorization: Bearer <access_token>`

**响应**
```json
{
  "success": true,
  "data": {
    "id":           1,
    "username":     "root",
    "display_name": "管理员",
    "role":         100,
    "status":       1,
    "email":        "admin@example.com",
    "quota":        500000,
    "used_quota":   12345,
    "request_count": 309,
    "group":        "default",
    "aff_code":     "xxxxx",
    "admin_permissions": {
      "channel": {
        "read": true,
        "write": true,
        "sensitive_write": true,
        "operate": true
      }
    }
  }
}
```

---

## OAuth 社交登录

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/oauth/state` | 生成 state code（防 CSRF） |
| GET | `/api/oauth/wechat` | 微信登录 |
| GET | `/api/oauth/telegram/login` | Telegram 登录 |
| GET | `/api/oauth/:provider` | GitHub / Discord / OIDC / LinuxDO |

provider 可选值：`github` | `discord` | `oidc` | `linuxdo`

---

## 错误响应格式

```json
{
  "success": false,
  "message": "错误描述"
}
```

认证错误附带 code 字段：
```json
{
  "success": false,
  "code":    "AUTH_TOKEN_EXPIRED",
  "message": "Token has expired"
}
```

**code 取值**

| code | 含义 |
|---|---|
| `AUTH_TOKEN_EXPIRED` | Token 已过期 |
| `AUTH_SESSION_REVOKED` | Session 已撤销 |
| `AUTH_UNAUTHORIZED` | 未认证 |
| `AUTH_INTERNAL_ERROR` | 内部错误 |
| `AUTH_USER_DISABLED` | 用户已禁用 |
| `AUTH_INSUFFICIENT_PRIVILEGE` | 权限不足 |
