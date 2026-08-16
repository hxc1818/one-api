# 添加 Codex Responses API 支持 - 实现清单

## 需求分析
支持 OpenAI Responses API (`/v1/responses`)，这是为 agentic 工作流设计的新端点，GPT-5-Codex 等模型仅在此 API 可用。

## 实现步骤

### 1. 后端核心逻辑
- [x] 1.1 在 `relay/relaymode/define.go` 添加 Responses 模式常量
- [x] 1.2 在 `relay/relaymode/helper.go` 添加 `/v1/responses` 路径识别
- [x] 1.3 在 `router/relay.go` 注册新路由 POST `/v1/responses`

### 2. 数据模型定义
- [x] 2.1 研究 Responses API 的完整请求格式
- [x] 2.2 在 `relay/model/` 中定义 ResponsesRequest 结构体
- [x] 2.3 定义 ResponsesResponse 响应结构体
- [x] 2.4 处理特殊字段（如 previous_response_id、modalities 等）

### 3. 控制器实现
- [x] 3.1 在 `relay/controller/` 创建 `responses.go` 文件
- [x] 3.2 实现 `RelayResponsesHelper` 函数（参考 RelayTextHelper）
- [x] 3.3 实现请求验证逻辑
- [x] 3.4 实现响应处理逻辑（包括流式和非流式）

### 4. 适配器支持
- [x] 4.1 在 `relay/adaptor/openai/` 中添加 Responses API 适配
- [x] 4.2 实现请求格式转换（直接透传）
- [x] 4.3 实现响应格式转换（使用现有逻辑）
- [ ] 4.4 处理 WebSocket 模式（可选，后期优化）

### 5. 计费和验证
- [x] 5.1 在 `relay/controller/validator/` 添加 Responses 请求验证
- [x] 5.2 实现 token 计数逻辑（可能需要特殊处理 agentic loop）
- [x] 5.3 配置模型倍率（如 GPT-5-Codex）

### 6. 主控制器集成
- [x] 6.1 在 `controller/relay.go` 的 relayHelper 中添加 Responses 分支
- [x] 6.2 确保错误处理和重试逻辑正确

### 7. 测试验证
- [x] 7.1 编译后端并启动服务
- [ ] 7.2 使用 curl 测试基本的 responses 请求
- [ ] 7.3 测试流式响应
- [ ] 7.4 测试工具调用场景
- [ ] 7.5 验证计费准确性

### 8. 文档更新
- [x] 8.1 更新 README.md 说明支持 Responses API
- [x] 8.2 添加使用示例和测试文档

---

## 注意事项
- Responses API 与 Chat Completions 不同，主要区别：
  - 支持 agentic loop（模型自主多次工具调用）
  - 支持 `previous_response_id` 实现增量输入
  - 支持 WebSocket 用于长时间运行的工作流
  - 工具调用方式不同

- 参考资源：
  - OpenAI Responses API 文档：https://platform.openai.com/docs/api-reference/responses
  - 现有实现模式：RelayTextHelper、RelayImageHelper

---

## 进度追踪
开始时间：已完成
预计完成时间：已完成
当前状态：✅ 基础实现已完成

## 实现总结

### 已完成的工作
1. ✅ 添加了 Responses 模式定义和路由识别
2. ✅ 注册了 `/v1/responses` 路由
3. ✅ 定义了完整的请求和响应数据模型
4. ✅ 实现了 RelayResponsesHelper 控制器
5. ✅ 添加了请求验证逻辑
6. ✅ 集成到主控制器的 relayHelper 中
7. ✅ 编译成功，无错误
8. ✅ 创建了详细的测试文档

### 技术细节
- **数据模型**：扩展了 `GeneralOpenAIRequest`，添加了 `InputItems`、`ConversationId`、`PreviousResponseId` 等字段
- **响应结构**：新增 `ResponsesAPIResponse`、`ResponseOutputItem`、`IncompleteDetails` 等类型
- **Token 计算**：实现了基于 `input_items` 的 token 估算逻辑
- **适配器**：复用现有的 OpenAI 适配器，支持直接透传
- **计费逻辑**：复用现有的计费系统，支持预消费和后消费

### 待测试功能
- 基本的 Responses API 请求
- 流式响应
- 工具调用场景
- conversation_id 连续对话
- 计费准确性

### 后续优化方向
- WebSocket 支持（用于长时间运行的工作流）
- 更精确的 token 计算（特别是 agentic loop）
- 更多参数支持（modalities 等）
- 性能优化和错误处理增强
