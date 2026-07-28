# NewAPI — Relay 网关接口

Base URL：`https://{your-newapi-instance}`（不加 `/api` 前缀）

认证：
- `Authorization: Bearer sk-<key>`（OpenAI 风格）
- `x-api-key: <key>`（Anthropic 风格）
- Query 参数 `?key=<key>`（Gemini 风格）

---

## OpenAI 兼容接口

### POST /v1/chat/completions

Chat Completions API，与 OpenAI 官方 API 完全兼容。

**请求体**（标准 OpenAI 格式）
```json
{
  "model":       "gpt-4o",
  "messages":    [{ "role": "user", "content": "Hello" }],
  "stream":      false,
  "max_tokens":  1024
}
```

### GET /v1/models

获取可用模型列表。

**响应**（OpenAI 格式）
```json
{
  "object": "list",
  "data": [
    { "id": "gpt-4o", "object": "model", "owned_by": "openai" }
  ]
}
```

### POST /v1/embeddings

文本嵌入接口。

### POST /v1/images/generations

图像生成接口。

### POST /v1/moderations

内容审核接口。

---

## Anthropic 兼容接口

### POST /v1/messages

Claude Messages API（通过 NewAPI 中转）。

**请求头**
```
Authorization: Bearer sk-your-key
anthropic-version: 2023-06-01
```

**请求体**（标准 Anthropic 格式）
```json
{
  "model":      "claude-opus-4-5",
  "max_tokens": 1024,
  "messages":   [{ "role": "user", "content": "Hello" }]
}
```

---

## Gemini 兼容接口

通过 Query 参数传递 key：

```
POST /v1beta/models/{model}:generateContent?key=sk-xxx
POST /v1beta/models/{model}:streamGenerateContent?key=sk-xxx
```

---

## 错误响应

**OpenAI 格式错误**
```json
{
  "error": {
    "message": "错误描述",
    "type":    "invalid_request_error",
    "code":    "model_not_found"
  }
}
```

**速率限制**：HTTP 429

**无效 Token**：HTTP 401

---

## 与 Sub2API 网关对比

| 维度 | Sub2API | NewAPI |
|---|---|---|
| OpenAI 兼容 | ✅ `/v1/*` | ✅ `/v1/*` |
| Anthropic 兼容 | ✅ `/v1/messages` | ✅ `/v1/messages` |
| Gemini 兼容 | ✅ `/v1beta/*` | ✅ `/v1beta/*` |
| 并发限制响应 | 429 + concurrency_limit_error | 429 |
| 流式输出 | ✅ SSE | ✅ SSE |
| 模型映射 | 渠道级 model_mapping | 渠道级 model_mapping |
