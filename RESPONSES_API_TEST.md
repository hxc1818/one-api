# Responses API 测试指南

## 准备工作

1. 启动 One API 服务：
```bash
./one-api --port 3000
```

2. 登录系统（默认用户名 root，密码 123456）

3. 在"渠道"页面添加 OpenAI 渠道，确保 API Key 有效

4. 在"令牌"页面创建访问令牌

## 测试用例

### 测试 1：基本的 Responses API 请求

```bash
curl -X POST http://localhost:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "gpt-4o",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Hello, what is the capital of France?"
          }
        ]
      }
    ],
    "max_tokens": 100
  }'
```

### 测试 2：流式响应

```bash
curl -X POST http://localhost:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "gpt-4o",
    "stream": true,
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Count from 1 to 10"
          }
        ]
      }
    ],
    "max_tokens": 150
  }'
```

### 测试 3：带工具调用的请求

```bash
curl -X POST http://localhost:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "gpt-4o",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "What is the weather in San Francisco?"
          }
        ]
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get the current weather in a location",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "The city and state"
              }
            },
            "required": ["location"]
          }
        }
      }
    ],
    "max_tokens": 200
  }'
```

### 测试 4：使用 conversation_id 的连续对话

```bash
# 第一次请求
curl -X POST http://localhost:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "gpt-4o",
    "conversation_id": "test-conversation-123",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "My name is Alice"
          }
        ]
      }
    ],
    "max_tokens": 100
  }'

# 第二次请求（使用相同的 conversation_id）
curl -X POST http://localhost:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "gpt-4o",
    "conversation_id": "test-conversation-123",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "What is my name?"
          }
        ]
      }
    ],
    "max_tokens": 100
  }'
```

## 验证要点

### 1. 路由验证
- [ ] `/v1/responses` 端点可以正常访问
- [ ] 返回正确的 HTTP 状态码（200 成功，400 参数错误，401 认证失败等）

### 2. 请求验证
- [ ] 缺少 `model` 字段时返回错误
- [ ] 缺少 `input_items` 字段时返回错误
- [ ] `max_tokens` 为负数时返回错误

### 3. 响应格式验证
- [ ] 非流式响应返回完整的 JSON 对象
- [ ] 流式响应正确返回 SSE 格式数据
- [ ] 响应包含 `usage` 信息（token 计数）

### 4. 计费验证
- [ ] 检查"日志"页面，确认请求被正确记录
- [ ] 验证 token 计数准确（prompt_tokens + completion_tokens）
- [ ] 确认用户额度正确扣减

### 5. 错误处理
- [ ] 无效的 API Key 返回 401
- [ ] 额度不足返回 403
- [ ] 上游 API 错误正确传递

## 预期响应格式

### 成功响应示例（非流式）
```json
{
  "id": "resp_abc123",
  "object": "response",
  "created": 1234567890,
  "model": "gpt-4o",
  "conversation_id": "test-conversation-123",
  "output_items": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {
          "type": "text",
          "text": "The capital of France is Paris."
        }
      ]
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 10,
    "total_tokens": 25
  },
  "status": "completed"
}
```

### 流式响应示例
```
data: {"id":"resp_abc123","object":"response.chunk","created":1234567890,...}

data: {"id":"resp_abc123","object":"response.chunk","delta":{"content":[{"type":"text","text":"The"}]},...}

data: {"id":"resp_abc123","object":"response.chunk","delta":{"content":[{"type":"text","text":" capital"}]},...}

...

data: [DONE]
```

## 故障排查

### 问题：404 Not Found
- 检查路由是否正确注册：`grep -r "responses" router/`
- 确认服务已重启

### 问题：500 Internal Server Error
- 查看日志文件：`tail -f logs/one-api.log`
- 检查上游 OpenAI API 是否可访问

### 问题：400 Bad Request
- 验证请求体格式是否正确
- 检查必需字段是否都提供了

### 问题：计费不准确
- 检查 `relay/controller/responses.go` 中的 token 计算逻辑
- 查看日志中的 usage 信息

## 注意事项

1. **模型可用性**：GPT-5-Codex 等新模型可能只在 Responses API 中可用
2. **API 兼容性**：确保上游 OpenAI 渠道支持 Responses API
3. **Token 计算**：Responses API 的 token 计算可能与 Chat Completions 不同，需要根据实际使用调整
4. **WebSocket 支持**：当前版本暂不支持 WebSocket，需要后续优化

## 下一步优化

- [ ] 添加 WebSocket 支持用于长时间运行的工作流
- [ ] 优化 token 计算逻辑，特别是 agentic loop 场景
- [ ] 添加更多的请求参数支持（modalities 等）
- [ ] 完善错误处理和重试机制
