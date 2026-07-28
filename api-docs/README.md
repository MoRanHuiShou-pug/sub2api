# API 文档总览

本目录记录 **Sub2API** 与 **NewAPI** 两套系统的管理接口，重点面向"上游管理"功能开发使用。

## 目录结构

```
api-docs/
├── README.md          本文件（概览 + 差异对比）
├── sub2api/
│   ├── 01-auth.md     认证接口（登录/刷新/OAuth）
│   ├── 02-accounts.md 账号管理（CRUD + 状态操作）
│   ├── 03-groups.md   分组管理（倍率 / 容量）
│   ├── 04-dashboard.md 仪表盘统计（健康状态）
│   └── 05-gateway.md  AI 网关（OpenAI / Anthropic 兼容）
└── newapi/
    ├── 01-auth.md     认证接口（登录/刷新/OAuth）
    ├── 02-channels.md 渠道管理（CRUD + 状态操作）
    ├── 03-groups.md   分组管理
    └── 04-gateway.md  Relay 网关（OpenAI / Anthropic 兼容）
```

---

## 核心差异对比

| 维度 | Sub2API | NewAPI |
|---|---|---|
| **Base URL 前缀** | `/api/v1/` | `/api/` |
| **登录路径** | `POST /api/v1/auth/login` | `POST /api/user/login` |
| **Token 类型** | JWT（Bearer） | JWT（Bearer）或 PAT |
| **刷新机制** | 显式 refresh_token（请求体传参） | HttpOnly Cookie（静默续期） |
| **账号/渠道** | Account（`/admin/accounts`） | Channel（`/api/channel/`） |
| **账号标识字段** | `platform` + `type` + `credentials` | `type`（int 枚举）+ `key` + `base_url` |
| **分组** | Group（`/admin/groups`） | Group（`/api/group/`） |
| **倍率字段** | `rate_multiplier`（float，账号级 + 分组级） | `weight`（uint，渠道权重）|
| **健康状态** | `status: active/error/rate_limited/overload` | `status: 1=启用/2=禁用/3=自动禁用` |
| **错误格式** | `{"code":0/非0, "message":"..."}` | `{"success":true/false, "message":"..."}` |
| **API Key Header** | `x-api-key` 或 `Authorization: Bearer` | `Authorization: Bearer sk-xxx` |

---

## 上游管理关键接口速查

### Sub2API（作为上游实例）

```
POST /api/v1/auth/login           登录，获取 access_token + refresh_token
POST /api/v1/auth/refresh         刷新 access_token
GET  /api/v1/admin/accounts       拉取账号列表（含状态、倍率、优先级）
GET  /api/v1/admin/groups         拉取分组列表（含 rate_multiplier）
GET  /api/v1/admin/dashboard/stats 仪表盘健康统计（error/normal/overload 账号数）
```

### NewAPI（作为上游实例）

```
POST /api/user/login              登录，获取 access_token（session）
POST /api/user/auth/refresh       刷新 access_token（Cookie）
GET  /api/channel/                拉取渠道列表（含状态、余额、权重）
GET  /api/group/                  拉取分组列表
GET  /api/status                  系统状态
```
