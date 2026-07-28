# Sub2API — 认证接口

Base URL：`https://{your-sub2api-instance}/api/v1`

所有需要认证的接口使用：
- `Authorization: Bearer <access_token>`（推荐）
- `x-api-key: <admin-api-key>`（管理员接口可用）

---

## POST /auth/login

邮箱密码登录，返回 JWT 令牌对。

**速率限制**：20 次/分钟

**请求体**
```json
{
  "email":           "admin@example.com",
  "password":        "your-password",
  "turnstile_token": "optional"
}
```

**响应 A — 登录成功**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token":  "eyJhbGci...",
    "refresh_token": "rt_xxx",
    "expires_in":    86400,
    "token_type":    "Bearer",
    "user": {
      "id":         1,
      "email":      "admin@example.com",
      "username":   "管理员",
      "role":       "admin",
      "balance":    997.72,
      "concurrency": 10,
      "status":     "active"
    }
  }
}
```

**响应 B — 需要 2FA**
```json
{
  "code": 0,
  "data": {
    "requires_2fa":     true,
    "temp_token":       "tmp_xxx",
    "user_email_masked": "ad***@example.com"
  }
}
```

---

## POST /auth/login/2fa

完成 2FA 验证，适用于登录返回 `requires_2fa: true` 的情况。

**速率限制**：20 次/分钟

**请求体**
```json
{
  "temp_token": "tmp_xxx",
  "totp_code":  "123456"
}
```

**响应**：同 `/auth/login` 的成功响应。

---

## POST /auth/refresh

使用 refresh_token 换取新的 access_token。access_token 有效期默认 24 小时，**建议在过期前 10 分钟主动刷新**。

**速率限制**：30 次/分钟

**请求体**
```json
{
  "refresh_token": "rt_xxx"
}
```

**响应**
```json
{
  "code": 0,
  "data": {
    "access_token":  "eyJhbGci...",
    "refresh_token": "rt_yyy",
    "expires_in":    86400,
    "token_type":    "Bearer"
  }
}
```

> ⚠️ 每次刷新会返回新的 refresh_token，旧的立即失效，请务必更新存储。

---

## POST /auth/logout

主动登出，可选撤销 refresh_token。

**请求头**：`Authorization: Bearer <access_token>`

**请求体**
```json
{
  "refresh_token": "rt_xxx"
}
```

**响应**
```json
{
  "code": 0,
  "data": { "message": "Logged out successfully" }
}
```

---

## GET /auth/me

获取当前登录用户的完整信息。

**请求头**：`Authorization: Bearer <access_token>`

**响应**
```json
{
  "code": 0,
  "data": {
    "id":            1,
    "email":         "admin@example.com",
    "username":      "管理员",
    "role":          "admin",
    "balance":       997.72,
    "concurrency":   10,
    "status":        "active",
    "allowed_groups": null,
    "last_active_at": "2026-07-28T10:00:00+08:00",
    "created_at":    "2026-06-09T16:15:41+08:00",
    "rpm_limit":     0,
    "balance_notify_enabled":          true,
    "balance_notify_threshold_type":   "fixed",
    "total_recharged": 0
  }
}
```

---

## POST /auth/send-verify-code

发送邮箱验证码（注册/忘记密码等前置步骤）。

**速率限制**：5 次/分钟

**请求体**
```json
{
  "email":   "user@example.com",
  "purpose": "register"
}
```

`purpose` 取值：`register` | `forgot-password`

**响应**
```json
{
  "code": 0,
  "data": { "message": "Verification code sent successfully", "countdown": 60 }
}
```

---

## POST /auth/register

注册新用户，需先调用 `/auth/send-verify-code`。

**速率限制**：5 次/分钟

**请求体**
```json
{
  "email":           "user@example.com",
  "password":        "password123",
  "verify_code":     "123456",
  "invitation_code": "optional"
}
```

**响应**：同 `/auth/login` 成功响应（注册后自动登录）。

---

## POST /auth/revoke-all-sessions

撤销当前用户所有活跃 session（安全操作，登出所有设备）。

**请求头**：`Authorization: Bearer <access_token>`

---

## OAuth 社交登录

各提供商的"发起 → 回调"流程统一如下：

| 提供商 | 发起 URL | 回调 URL |
|---|---|---|
| GitHub | `GET /auth/oauth/github/start` | `GET /auth/oauth/github/callback` |
| Google | `GET /auth/oauth/google/start` | `GET /auth/oauth/google/callback` |
| LinuxDo | `GET /auth/oauth/linuxdo/start` | `GET /auth/oauth/linuxdo/callback` |
| WeChat | `GET /auth/oauth/wechat/start` | `GET /auth/oauth/wechat/callback` |
| DingTalk | `GET /auth/oauth/dingtalk/start` | `GET /auth/oauth/dingtalk/callback` |
| OIDC | `GET /auth/oauth/oidc/start` | `GET /auth/oauth/oidc/callback` |

OAuth Pending 统一二步完成（跨提供商）：
```
POST /auth/oauth/pending/exchange
POST /auth/oauth/pending/send-verify-code
POST /auth/oauth/pending/create-account
POST /auth/oauth/pending/bind-login
```

---

## Passkey 认证

**无需认证（登录流程）**
```
POST /auth/passkey/login/begin    速率限制: 20次/分钟
POST /auth/passkey/login/finish   速率限制: 20次/分钟
```

**需要认证（注册/管理）**
```
GET    /user/passkeys
POST   /user/passkeys/register/begin
POST   /user/passkeys/register/finish
PATCH  /user/passkeys/:id
DELETE /user/passkeys/:id
```

---

## 错误码

| code | 含义 |
|---|---|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 / Token 无效或已过期 |
| 403 | 权限不足 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
