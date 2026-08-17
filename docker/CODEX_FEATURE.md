# Codex 密钥类型功能说明

## 功能概述

新增了 "Codex 格式" 密钥类型，用于自动将 OpenAI Responses API (`/v1/responses`) 请求转换为 Chat Completions API (`/v1/chat/completions`) 请求。

这个功能主要解决的问题是：当渠道不支持 `/v1/responses` 端点时，用户仍然可以通过使用 Codex 格式的密钥来访问服务。

## 工作原理

### 1. 请求流程

当用户使用 Codex 类型的密钥请求 `/v1/responses` 时：

1. **中间件验证**：系统识别出密钥类型为 "codex"
2. **路由转换**：自动将请求模式从 `Responses` 转换为 `ChatCompletions`
3. **请求格式转换**：将 `input_items` 转换为 `messages`
4. **发送到渠道**：使用标准的 `/v1/chat/completions` 端点
5. **响应格式转换**：将 Chat Completions 的流式响应转换为 Responses API 格式
6. **返回给用户**：用户收到符合 Responses API 格式的响应

### 2. 格式转换细节

#### 请求转换 (input_items → messages)

```javascript
// 输入 (Responses API)
{
  "model": "gpt-4",
  "input_items": [
    {
      "type": "message",
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Hello"
        }
      ]
    }
  ]
}

// 转换后 (Chat Completions API)
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "Hello"
    }
  ]
}
```

#### 响应转换 (Chat Stream → Responses Stream)

```
# Chat Completions SSE:
data: {"id":"chatcmpl-123","choices":[{"delta":{"content":"Hi"}}]}

# 转换为 Responses API SSE:
event: response.output_item.added
data: {"type":"response.output_item.added","item":{...}}

event: response.output_item.delta
data: {"type":"response.output_item.delta","delta":{"text":"Hi"}}

event: response.output_item.done
data: {"type":"response.output_item.done"}

event: response.done
data: {"type":"response.done","status":"completed"}
```

## 使用方法

### 1. 创建 Codex 类型密钥

1. 登录 One-API 管理后台
2. 进入 "令牌" 页面
3. 点击 "新建令牌"
4. 在 "密钥类型" 下拉框中选择 "Codex 格式 (自动转换 /v1/responses)"
5. 填写其他必要信息后保存

### 2. 使用密钥

```bash
curl https://your-one-api-domain/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-codex-key" \
  -d '{
    "model": "gpt-4",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Say hello!"
          }
        ]
      }
    ],
    "stream": true
  }'
```

系统会自动：
- 将请求转换为 Chat Completions 格式
- 发送到支持的渠道
- 将响应转换回 Responses API 格式
- 返回给用户

## 计费说明

- 计费基于实际的 **Chat Completions API** 调用
- Token 消耗按照实际使用的 prompt tokens 和 completion tokens 计算
- 预消费和后消费逻辑与普通 Chat Completions 请求一致

## 技术实现

### 修改的文件

1. **common/ctxkey/key.go** - 添加上下文键
2. **middleware/auth.go** - 保存 token 类型到上下文
3. **controller/relay.go** - 路由判断和转换逻辑
4. **relay/controller/text.go** - 请求转换
5. **relay/adaptor/openai/adaptor.go** - 响应处理
6. **relay/adaptor/codex/converter.go** - 格式转换逻辑
7. **relay/adaptor/codex/stream.go** - 流式响应转换
8. **relay/model/message.go** - 消息模型扩展
9. **web/air/src/pages/Token/EditToken.js** - 前端界面（Air 主题）
10. **web/default/src/pages/Token/EditToken.js** - 前端界面（Default 主题）

### 核心组件

- `codex.InputItemsToMessages()` - 将 input_items 转换为 messages
- `codex.StreamHandlerWithCodexConversion()` - 处理流式响应转换
- 上下文标记 `need_codex_conversion` - 标识需要转换的请求

## 注意事项

1. **仅支持流式请求**：当前实现主要针对流式响应（`stream: true`）
2. **渠道兼容性**：渠道必须支持 `/v1/chat/completions` 端点
3. **功能限制**：某些 Responses API 的高级功能可能无法完全转换
4. **性能影响**：格式转换会增加少量延迟（通常 < 10ms）

## 测试建议

1. 创建一个 Codex 类型的密钥
2. 使用 Responses API 格式发送请求
3. 验证响应格式是否正确
4. 检查计费是否准确

## 故障排查

如果遇到问题：

1. 检查密钥类型是否设置为 "codex"
2. 确认渠道支持 Chat Completions API
3. 查看日志中的转换信息
4. 验证请求格式是否符合 Responses API 规范

## 未来改进

- [ ] 支持非流式响应转换
- [ ] 支持更多 Responses API 特性（如 conversation_id）
- [ ] 性能优化
- [ ] 添加详细的转换日志
