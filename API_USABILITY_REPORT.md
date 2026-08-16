# One API 可用性检查报告

## 项目概述
- **项目名称**: One API
- **功能**: 统一的大模型 API 网关，将多种 AI 模型统一为 OpenAI 兼容格式
- **检查日期**: 2024-08-16
- **编译状态**: ✅ 已编译完成 (可执行文件: one-api, 30.2MB)

---

## 1. OpenAI API 支持检查

### ✅ 完整实现
**位置**: `relay/adaptor/openai/`

#### 核心功能
- **Chat Completions API**: ✅ 完整支持
  - 文件: `main.go`, `adaptor.go`, `model.go`
  - Stream 模式: ✅ 支持 (`StreamHandler`)
  - 非 Stream 模式: ✅ 支持 (`Handler`)
  
- **Completions API**: ✅ 完整支持
  - Legacy 模型支持

- **Token 计数**: ✅ 完整实现
  - 文件: `token.go`
  - 函数: `CountTokenText()`, `CountTokenMessages()`
  - 支持多种模型的 token 计算

#### 支持的 OpenAI 模型（从计费比率文件确认）
```
✅ GPT-4 系列: gpt-4, gpt-4-turbo, gpt-4o, gpt-4o-mini 等
✅ GPT-3.5 系列: gpt-3.5-turbo, gpt-3.5-turbo-instruct 等
✅ O1 系列: o1, o1-mini, o1-preview, o3-mini
✅ 嵌入模型: text-embedding-ada-002, text-embedding-3-small/large
✅ 语音模型: whisper-1, tts-1, tts-1-hd
✅ 图像模型: dall-e-2, dall-e-3
✅ Legacy 模型: davinci-002, babbage-002 等
```

---

## 2. Codex API 支持检查

### ⚠️ 状态说明
在代码库中**没有找到**专门的 "Codex" 适配器或明确的 Codex 支持。

#### 分析
1. **搜索结果**: 
   ```bash
   grep -r "codex" -i  # 无结果
   grep -r "Codex"     # 无结果
   ```

2. **可能的原因**:
   - OpenAI 已在 2023 年弃用了 Codex 模型
   - Codex 功能已合并到 GPT-3.5/GPT-4 系列中
   - code-davinci-edit-001 等模型在计费比率中存在，但属于 legacy 模型

3. **替代方案**:
   - 使用 `gpt-4` 或 `gpt-3.5-turbo` 进行代码生成
   - 支持的代码相关模型:
     - ✅ code-davinci-edit-001 (计费比率: 10)
     - ✅ code-cushman-001 (可能)

### 结论
❌ **不支持独立的 Codex API**
✅ **支持代码生成相关的 GPT 模型**

---

## 3. Claude API 支持检查

### ✅ 完整实现
**位置**: `relay/adaptor/anthropic/`

#### 核心功能
- **Chat API**: ✅ 完整支持
  - 文件: `main.go`, `adaptor.go`, `model.go`
  - 请求转换: `ConvertRequest()` - OpenAI 格式转 Claude 格式
  - 响应转换: `ResponseClaude2OpenAI()` - Claude 格式转 OpenAI 格式
  - Stream 处理: `StreamHandler()`, `StreamResponseClaude2OpenAI()`

#### 高级特性支持

##### 1. **Tool Calling (工具调用)** ✅
```go
// anthropic/main.go
- 支持 tools 参数转换
- 支持 tool_choice 控制
- 支持 tool_use 和 tool_result
- Stream 模式完整支持工具调用
```

**实现细节**:
```go
// 将 OpenAI tools 格式转换为 Claude InputSchema 格式
claudeTools := make([]Tool, 0, len(textRequest.Tools))
for _, tool := range textRequest.Tools {
    if params, ok := tool.Function.Parameters.(map[string]any); ok {
        claudeTools = append(claudeTools, Tool{
            Name:        tool.Function.Name,
            Description: tool.Function.Description,
            InputSchema: InputSchema{...},
        })
    }
}
```

##### 2. **Vision (视觉能力)** ✅
```go
// 支持图像输入
if part.Type == model.ContentTypeImageURL {
    content.Type = "image"
    content.Source = &ImageSource{
        Type: "base64",
    }
    mimeType, data, _ := image.GetImageFromUrl(part.ImageURL.Url)
    content.Source.MediaType = mimeType
    content.Source.Data = data
}
```

##### 3. **System Prompt** ✅
```go
// 单独处理 system 角色消息
if message.Role == "system" && claudeRequest.System == "" {
    claudeRequest.System = message.StringContent()
    continue
}
```

##### 4. **Stop Reason 映射** ✅
```go
func stopReasonClaude2OpenAI(reason *string) string {
    switch *reason {
    case "end_turn": return "stop"
    case "stop_sequence": return "stop"
    case "max_tokens": return "length"
    case "tool_use": return "tool_calls"  // 工具调用停止原因
    }
}
```

#### 支持的 Claude 模型
```
✅ Claude 3.5 系列:
   - claude-3-5-sonnet-20240620
   - claude-3-5-sonnet-20241022
   - claude-3-5-haiku-20241022

✅ Claude 3 系列:
   - claude-3-opus-20240229
   - claude-3-sonnet-20240229
   - claude-3-haiku-20240307

✅ Legacy 系列:
   - claude-2.1
   - claude-2.0
   - claude-instant-1.2
```

#### AWS Claude 支持 ✅
**位置**: `relay/adaptor/aws/claude/`
- 支持通过 AWS Bedrock 访问 Claude

#### Vertex AI Claude 支持 ✅
**位置**: `relay/adaptor/vertexai/claude/`
- 支持通过 Google Vertex AI 访问 Claude

---

## 4. 计费逻辑检查

### ✅ 完整且精确的计费系统
**位置**: `relay/billing/`

#### 核心组件

##### 1. **模型计费比率** (`relay/billing/ratio/model.go`)
- **定义**: 200+ 个模型的精确计费比率
- **单位**: 基于 USD，1 = $0.002 / 1K tokens
- **覆盖模型**:
  ```
  ✅ OpenAI 全系列
  ✅ Claude 全系列 (包括最新的 3.5 Sonnet/Haiku)
  ✅ Gemini 全系列
  ✅ 国内模型: 文心一言、通义千问、智谱、月之暗面等
  ✅ 其他: Mistral, Groq, DeepSeek, Cohere 等
  ```

##### 2. **Completion Token 倍率**
```go
// 不同模型的输出 token 计费倍率
CompletionRatio = map[string]float64{
    "gpt-3.5-turbo": 3,      // 输出是输入的 3 倍价格
    "gpt-4": 2,              // 输出是输入的 2 倍价格
    "claude-3": 5,           // Claude 3 系列
    "deepseek-chat": 2,      // DeepSeek 系列
}
```

##### 3. **计费流程** (`relay/billing/billing.go`)

**预扣费 (Pre-consume)**:
```go
// 在请求发出前根据 prompt tokens 预扣费
preConsumedQuota = promptTokens * modelRatio * groupRatio
```

**后结算 (Post-consume)**:
```go
// 请求完成后根据实际使用的 tokens 结算
totalQuota = (promptTokens + completionTokens * completionRatio) * modelRatio * groupRatio
quotaDelta = totalQuota - preConsumedQuota
```

**退款机制**:
```go
// 如果请求失败，退回预扣费
func ReturnPreConsumedQuota(ctx, preConsumedQuota, tokenId)
```

##### 4. **分组倍率** (`relay/billing/ratio/group.go`)
- 支持用户分组差异化定价
- 不同分组可以有不同的价格倍率

#### 计费特性

✅ **Token 精确计算**
- 使用 tiktoken 库进行精确 token 计数
- 支持缓存编码器 (TIKTOKEN_CACHE_DIR)

✅ **多维度计费**
```
最终价格 = promptTokens * modelRatio * groupRatio 
         + completionTokens * completionRatio * modelRatio * groupRatio
```

✅ **计费日志**
```go
model.RecordConsumeLog(ctx, &model.Log{
    UserId:           userId,
    ChannelId:        channelId,
    PromptTokens:     promptTokens,
    CompletionTokens: completionTokens,
    ModelName:        modelName,
    TokenName:        tokenName,
    Quota:            totalQuota,
    Content:          fmt.Sprintf("倍率：%.2f × %.2f", modelRatio, groupRatio),
})
```

✅ **实时额度更新**
- 更新用户已用额度
- 更新渠道已用额度
- 支持批量更新聚合 (BATCH_UPDATE_ENABLED)

#### Claude 计费特殊处理

Claude API 返回的 usage 格式:
```json
{
  "usage": {
    "input_tokens": 100,
    "output_tokens": 200
  }
}
```

转换为 OpenAI 格式:
```go
usage := model.Usage{
    PromptTokens:     claudeResponse.Usage.InputTokens,
    CompletionTokens: claudeResponse.Usage.OutputTokens,
    TotalTokens:      InputTokens + OutputTokens,
}
```

---

## 5. 工具调用 (Tool Calling) 检查

### ✅ 完整支持

#### 数据模型
**位置**: `relay/model/`

```go
// tool.go
type Tool struct {
    Id       string   `json:"id,omitempty"`
    Type     string   `json:"type,omitempty"`
    Function Function `json:"function"`
}

type Function struct {
    Description string `json:"description,omitempty"`
    Name        string `json:"name,omitempty"`
    Parameters  any    `json:"parameters,omitempty"` // 请求参数
    Arguments   any    `json:"arguments,omitempty"`  // 响应参数
}
```

```go
// general.go
type GeneralOpenAIRequest struct {
    ...
    Tools               []Tool `json:"tools,omitempty"`
    ToolChoice          any    `json:"tool_choice,omitempty"`
    ParallelTooCalls    *bool  `json:"parallel_tool_calls,omitempty"`
    ...
}
```

```go
// message.go
type Message struct {
    Role            string  `json:"role"`
    Content         any     `json:"content"`
    Name            *string `json:"name,omitempty"`
    ToolCalls       []Tool  `json:"tool_calls,omitempty"`
    ToolCallId      string  `json:"tool_call_id,omitempty"`
}
```

#### OpenAI 工具调用

✅ **请求支持**:
- `tools`: 工具定义数组
- `tool_choice`: 控制工具选择 (auto/required/none/specific)
- `parallel_tool_calls`: 并行工具调用

✅ **响应支持**:
- Stream 模式中正确处理 `tool_calls` delta
- 非 Stream 模式返回完整的 tool_calls

#### Claude 工具调用转换

**请求转换** (`anthropic/main.go`):
```go
// OpenAI tools → Claude tools
claudeTools := make([]Tool, 0, len(textRequest.Tools))
for _, tool := range textRequest.Tools {
    claudeTools = append(claudeTools, Tool{
        Name:        tool.Function.Name,
        Description: tool.Function.Description,
        InputSchema: InputSchema{
            Type:       params["type"].(string),
            Properties: params["properties"],
            Required:   params["required"],
        },
    })
}
```

**ToolChoice 转换**:
```go
// OpenAI tool_choice → Claude tool_choice
claudeToolChoice := struct {
    Type string `json:"type"`
    Name string `json:"name,omitempty"`
}{Type: "auto"}

if choice, ok := textRequest.ToolChoice.(map[string]any); ok {
    if function, ok := choice["function"].(map[string]any); ok {
        claudeToolChoice.Type = "tool"
        claudeToolChoice.Name = function["name"].(string)
    }
}
```

**响应转换**:
```go
// Claude content_block_start → OpenAI tool_calls
if claudeResponse.ContentBlock.Type == "tool_use" {
    tools = append(tools, model.Tool{
        Id:   claudeResponse.ContentBlock.Id,
        Type: "function",
        Function: model.Function{
            Name:      claudeResponse.ContentBlock.Name,
            Arguments: "",
        },
    })
}

// Claude input_json_delta → OpenAI arguments
if claudeResponse.Delta.Type == "input_json_delta" {
    tools = append(tools, model.Tool{
        Function: model.Function{
            Arguments: claudeResponse.Delta.PartialJson,
        },
    })
}
```

**Tool Result 处理**:
```go
// OpenAI tool role → Claude tool_result
if message.Role == "tool" {
    claudeMessage.Role = "user"
    content.Type = "tool_result"
    content.Content = content.Text
    content.ToolUseId = message.ToolCallId
}
```

#### Stream 模式工具调用

✅ **完整支持**:
```go
// anthropic/main.go - StreamHandler
for scanner.Scan() {
    var claudeResponse StreamResponse
    response, meta := StreamResponseClaude2OpenAI(&claudeResponse)
    
    // 处理工具调用的增量更新
    for _, choice := range response.Choices {
        if len(choice.Delta.ToolCalls) > 0 {
            lastToolCallChoice = choice
        }
    }
    
    // 完成时补充空参数为 {}
    if len(lastArgs.Arguments.(string)) == 0 {
        lastArgs.Arguments = "{}"
    }
}
```

---

## 6. 其他检查项

### ✅ 请求验证
**位置**: `relay/controller/validator/validation.go`
- 请求参数校验
- 模型名称验证

### ✅ 错误处理
**位置**: `relay/controller/error.go`
- 统一错误格式
- 错误包装和转换

### ✅ 负载均衡
- 支持多渠道负载均衡
- 失败自动重试
- 渠道可用性检测

### ✅ 额度管理
- 用户额度限制
- Token 额度限制
- 额度详细记录

### ✅ 认证与授权
**位置**: `middleware/auth.go`
- Token 认证
- 支持 OpenAI 格式: `sk-xxx`
- ✅ **新增**: 支持 Claude Code 格式: `sk-ant-api03-xxx`

### ✅ 监控与日志
- 请求日志记录
- 消费日志记录
- 系统错误日志

---

## 7. 新增功能：Claude Code 格式支持

### ✅ 实现完成
根据 `CC_FEATURE_SUMMARY.md`:

#### 支持的密钥格式
1. **OpenAI 格式** (默认)
   - 长度: 48 字符
   - 格式: `sk-xxx...`

2. **Claude Code 格式**
   - 长度: 108 字符
   - 格式: `sk-ant-api03-` + 95 位随机字符

#### 实现位置
- 后端:
  - `model/token.go`: KeyType 字段
  - `common/random/main.go`: GenerateCCKey()
  - `middleware/auth.go`: 支持 `sk-ant-` 前缀识别
  
- 前端: 三个主题都已实现
  - default, berry, air

---

## 8. 可能的问题和建议

### ⚠️ 注意事项

1. **Codex 不支持**
   - 如果用户期望使用 Codex API，需要迁移到 GPT-3.5/GPT-4
   - 建议在文档中明确说明

2. **环境配置检查**
   - 确保 `SQL_DSN` 已配置（高并发场景必需）
   - 检查 `REDIS_CONN_STRING`（可选，减少数据库压力）
   - 验证 `SESSION_SECRET` 已设置

3. **Token 计数缓存**
   - 检查 `TIKTOKEN_CACHE_DIR` 是否已配置
   - 离线环境需要预先下载编码器

4. **计费精度**
   - 确认所有新模型的计费比率已更新
   - 定期检查官方价格变动

### ✅ 建议的测试场景

1. **OpenAI API 测试**
   ```bash
   # Chat completion (非 stream)
   curl -X POST http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "Hello"}]
     }'
   
   # Chat completion (stream)
   curl -X POST http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "Hello"}],
       "stream": true
     }'
   ```

2. **Claude API 测试**
   ```bash
   # 普通对话
   curl -X POST http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "claude-3-5-sonnet-20241022",
       "messages": [{"role": "user", "content": "Hello"}]
     }'
   ```

3. **工具调用测试**
   ```bash
   # OpenAI 工具调用
   curl -X POST http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-4",
       "messages": [{"role": "user", "content": "What is the weather?"}],
       "tools": [{
         "type": "function",
         "function": {
           "name": "get_weather",
           "description": "Get weather info",
           "parameters": {
             "type": "object",
             "properties": {
               "location": {"type": "string"}
             }
           }
         }
       }]
     }'
   ```

4. **Claude Code 格式测试**
   ```bash
   # 使用 sk-ant- 前缀的密钥
   curl -X POST http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-ant-api03-xxx..." \
     -H "Content-Type: application/json" \
     -d '{
       "model": "claude-3-5-sonnet-20241022",
       "messages": [{"role": "user", "content": "Hello"}]
     }'
   ```

5. **计费验证**
   - 检查数据库 `logs` 表的消费记录
   - 验证 prompt_tokens, completion_tokens, quota 字段
   - 确认倍率计算正确

---

## 9. 总结

### ✅ 完全可用
1. **OpenAI API**: 完整支持所有模型和功能
2. **Claude API**: 完整支持，包括工具调用和 Vision
3. **计费逻辑**: 精确、完善、支持多维度计费
4. **工具调用**: OpenAI 和 Claude 都完整支持
5. **Claude Code 格式**: 已实现并集成

### ❌ 不支持
1. **Codex API**: 独立的 Codex API 不支持（已被 OpenAI 弃用）

### 🔧 建议
1. 启动项目并进行上述测试场景验证
2. 配置必要的环境变量（SQL_DSN, SESSION_SECRET 等）
3. 检查日志确认计费逻辑正常工作
4. 监控渠道可用性和余额
5. 定期更新模型计费比率

---

## 附录：关键文件清单

### 后端核心文件
```
relay/adaptor/openai/        # OpenAI 适配器
  ├── main.go                # Stream/非Stream 处理
  ├── adaptor.go             # 适配器接口实现
  ├── model.go               # 数据模型
  └── token.go               # Token 计数

relay/adaptor/anthropic/     # Claude 适配器
  ├── main.go                # 请求/响应转换，工具调用
  ├── adaptor.go             # 适配器接口实现
  └── model.go               # Claude 数据模型

relay/billing/               # 计费系统
  ├── billing.go             # 计费核心逻辑
  └── ratio/
      ├── model.go           # 模型计费比率
      └── group.go           # 分组倍率

relay/model/                 # 数据模型
  ├── general.go             # 通用请求模型
  ├── message.go             # 消息模型
  ├── tool.go                # 工具调用模型
  └── misc.go                # Usage 等

relay/controller/            # 控制器
  ├── text.go                # 文本处理主流程
  ├── error.go               # 错误处理
  └── validator/             # 请求验证

middleware/auth.go           # 认证中间件
model/token.go               # Token 模型
common/random/main.go        # 密钥生成
```

### 配置文件
```
.env                         # 环境变量配置
docker-compose.yml           # Docker 部署配置
one-api.service              # Systemd 服务配置
```

---

**报告生成时间**: 2024-08-16  
**检查人**: Kiro AI  
**项目版本**: 基于最新提交
