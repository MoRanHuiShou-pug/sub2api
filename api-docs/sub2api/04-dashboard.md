# Sub2API — 仪表盘统计接口

Base URL：`https://{your-sub2api-instance}/api/v1`

认证：`Authorization: Bearer <access_token>`（需要 admin 角色）

---

## GET /admin/dashboard/stats

**获取系统健康总览**。上游管理同步时最核心的健康探针接口。

**响应**
```json
{
  "code": 0,
  "data": {
    "total_accounts":      3,
    "normal_accounts":     3,
    "error_accounts":      0,
    "ratelimit_accounts":  0,
    "overload_accounts":   0,

    "total_users":         5,
    "today_new_users":     2,
    "active_users":        3,
    "hourly_active_users": 1,

    "total_api_keys":      6,
    "active_api_keys":     6,

    "total_requests":      309,
    "today_requests":      289,
    "total_tokens":        10134524,
    "today_tokens":        9493638,
    "total_input_tokens":  7188278,
    "today_input_tokens":  7112264,
    "total_output_tokens": 135254,
    "today_output_tokens": 122766,
    "total_cache_read_tokens":     2810992,
    "today_cache_read_tokens":     2258608,
    "total_cache_creation_tokens": 0,
    "today_cache_creation_tokens": 0,

    "total_cost":          41.11,
    "today_cost":          40.14,
    "total_actual_cost":   4.54,
    "today_actual_cost":   3.57,

    "average_duration_ms": 11170.5,
    "tpm":                 71636,
    "uptime":              113432,

    "stats_updated_at":    "2026-06-10T15:45:41Z",
    "stats_stale":         false
  }
}
```

**账号健康字段说明**

| 字段 | 含义 |
|---|---|
| `normal_accounts` | 正常可调度的账号数 |
| `error_accounts` | 处于 error 状态的账号数（登录失效、API 报错等） |
| `ratelimit_accounts` | 触发速率限制（429）暂时不可用的账号数 |
| `overload_accounts` | 触发过载（529）暂时不可用的账号数 |
| `total_accounts` | 所有账号总数 |

**健康分计算建议**
```
健康分 = normal_accounts / total_accounts × 100
若 error_accounts / total_accounts > 0.5 → 视为严重降级
```

---

## GET /admin/dashboard/snapshot-v2

综合快照（含趋势、账号分布、模型分布等），数据量较大，按需调用。

---

## GET /admin/dashboard/realtime

实时流量数据（当前并发、TPM 等），用于监控展示。

---

## GET /admin/dashboard/trend

时序趋势数据（请求量、Token 消耗按小时/天分组）。

**查询参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `start` | string | 开始时间（ISO 8601） |
| `end` | string | 结束时间（ISO 8601） |
| `granularity` | string | `hour` \| `day` |

---

## GET /admin/dashboard/groups

各分组使用情况统计（请求数、Token 数、费用按分组汇总）。

---

## GET /admin/dashboard/models

模型使用分布（各模型调用占比）。

---

## GET /admin/ops/account-availability

账号可用性实时检测，返回各账号的调度状态快照。
