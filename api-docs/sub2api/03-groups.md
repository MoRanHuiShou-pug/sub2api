# Sub2API — 分组管理接口

Base URL：`https://{your-sub2api-instance}/api/v1`

认证：`Authorization: Bearer <access_token>`（需要 admin 角色）

---

## GET /admin/groups

获取分组列表（分页）。

**响应**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id":               3,
        "name":             "DeepSeek - 官",
        "description":      "opus = deepseek-v4-pro\nsonnet = deepseek-v4-flash",
        "platform":         "anthropic",
        "status":           "active",
        "rate_multiplier":  1.0,
        "is_exclusive":     false,
        "subscription_type":"standard",
        "account_count":    1,
        "active_account_count":       1,
        "rate_limited_account_count": 0,
        "rpm_limit":        0,
        "daily_limit_usd":  null,
        "sort_order":       0
      }
    ],
    "total": 4
  }
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|---|---|---|
| `rate_multiplier` | float64 | 分组计费倍率（与账号倍率相乘得最终费率）。1.0 = 原价，2.0 = 双倍计费 |
| `account_count` | int | 该分组内账号总数 |
| `active_account_count` | int | 正常可用的账号数 |
| `rate_limited_account_count` | int | 触发速率限制的账号数 |
| `rpm_limit` | int | 分组级 RPM 上限（0 = 不限） |
| `is_exclusive` | bool | 是否独占分组（不参与轮询） |

---

## GET /admin/groups/all

获取所有分组（含已禁用），不分页。适合下拉选择器使用。

---

## GET /admin/groups/:id

获取单个分组详情（含模型路由配置）。

---

## POST /admin/groups

创建新分组。

**请求体**
```json
{
  "name":            "分组名称",
  "platform":        "anthropic",
  "description":     "描述信息",
  "rate_multiplier": 1.0,
  "status":          "active",
  "rpm_limit":       0,
  "daily_limit_usd": null
}
```

---

## PUT /admin/groups/:id

更新分组（全量覆盖）。

---

## DELETE /admin/groups/:id

删除分组。

---

## GET /admin/groups/capacity-summary

**分组容量汇总**。包含每个分组当前的账号健康状态分布，是上游同步中判断分组可用性的核心接口。

**响应示例**
```json
{
  "code": 0,
  "data": {
    "groups": [
      {
        "id":                         3,
        "name":                       "DeepSeek - 官",
        "platform":                   "anthropic",
        "total_accounts":             1,
        "normal_accounts":            1,
        "error_accounts":             0,
        "rate_limited_accounts":      0,
        "overload_accounts":          0,
        "schedulable_accounts":       1,
        "current_concurrency":        0,
        "max_concurrency":            10
      }
    ]
  }
}
```

---

## GET /admin/groups/usage-summary

分组使用量汇总（请求数、Token 数、费用按分组汇总）。

---

## GET /admin/groups/:id/rate-multipliers

获取分组内各账号的倍率配置（用于精细化计费）。

## PUT /admin/groups/:id/rate-multipliers

批量设置分组内各账号的倍率。

**请求体**
```json
{
  "multipliers": {
    "12": 1.5,
    "13": 2.0
  }
}
```

## DELETE /admin/groups/:id/rate-multipliers

清除分组内所有账号的倍率覆盖（恢复默认）。

---

## PUT /admin/groups/sort-order

调整分组排序。

**请求体**
```json
{
  "group_ids": [3, 4, 5]
}
```

---

## GET /admin/groups/:id/stats

获取分组使用统计（按时间范围）。
