# Sub2API — AI 网关接口

Base URL：`https://{your-sub2api-instance}`（注意：网关接口**不加** `/api/v1` 前缀）

认证：`Authorization: Bearer <api-key>` 或 `x-api-key: <api-key>`

---

## Anthropic 兼容接口

### POST /v1/messages

Claude Messages API，与 Anthropic 官方 API 完全兼容。

**请求头**
```
Authorization: Bearer sk-your-api-key
anthropic-version: 2023-06-01
content-type: application/json
```

**请求体**（标准 Anthropic 格式）
```json
{
  "model":      "claude-opus-4-5",
  "max_tokens": 1024,
  "messages": [
    { "role": "user", "content": "Hello" }
  ]
}
```

**流式响应**：附加 `"stream": true` 参数，返回 `text/event-stream`。

---

## OpenAI 兼容接口

### POST /v1/chat/completions

OpenAI Chat Completions API。

**请求头**
```
Authorization: Bearer sk-your-api-key
Content-Type: application/json
```

**请求体**（标准 OpenAI 格式）
```json
{
  "model":    "gpt-4o",
  "messages": [{ "role": "user", "content": "Hello" }],
  "stream":   false
}
```

### GET /v1/models

获取可用模型列表（OpenAI 格式）。

### POST /v1/embeddings

文本嵌入接口。

### POST /v1/images/generations

图像生成接口。

### POST /v1/audio/speech

TTS 语音合成。

---

## Gemini 兼容接口

### POST /v1beta/models/{model}:generateContent

Gemini GenerateContent API。

### POST /v1beta/models/{model}:streamGenerateContent

流式生成。

---

## 通用说明

**API Key 格式**：`sk-` 开头的字符串（用户在面板生成）

**错误响应（OpenAI 格式）**
```json
{
  "error": {
    "message": "错误描述",
    "type":    "invalid_request_error",
    "code":    "model_not_found"
  }
}
```

**速率限制响应**：HTTP 429，同上格式，`type` 为 `"rate_limit_error"`。

**并发超限响应**：HTTP 429，`type` 为 `"concurrency_limit_error"`。
