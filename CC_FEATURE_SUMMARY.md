# CC (Claude Code) 格式支持 - 完成报告

## 功能概述
为 one-api 项目添加了 Claude Code (CC) 格式密钥支持，用户现在可以在创建密钥时选择生成 OpenAI 兼容格式或 CC 格式的密钥。

## 实现细节

### 后端改动

1. **数据模型** (`model/token.go`)
   - 添加 `KeyType` 字段：存储密钥类型 ('openai' 或 'cc')
   - 修改 `Key` 字段：从 `char(48)` 改为 `varchar(150)` 以支持更长的 CC 格式密钥
   - 更新 `Update()` 方法：包含 `key_type` 字段

2. **密钥生成** (`common/random/main.go`)
   - 新增 `GenerateCCKey()` 函数：生成 CC 格式密钥 (`sk-ant-api03-` + 95位随机字符)
   - 保留原有 `GenerateKey()` 函数：生成 48位随机密钥

3. **控制器** (`controller/token.go`)
   - 新增 `generateKeyByType()` 辅助函数：根据 key_type 调用相应的生成函数
   - 修改 `AddToken()`：使用 key_type 生成对应格式的密钥
   - 修改 `UpdateToken()`：支持更新 key_type 字段

4. **认证中间件** (`middleware/auth.go`)
   - 修改 `TokenAuth()`：支持识别和去除 `sk-ant-` 前缀
   - 处理顺序：`Bearer ` → `sk-ant-` → `sk-`

### 前端改动

1. **default 主题** (`web/default/`)
   - `EditToken.js`：添加密钥类型下拉选择器（创建时可选，编辑时禁用）
   - `TokensTable.js`：添加"密钥类型"列，使用 Label 标签显示

2. **berry 主题** (`web/berry/`)
   - `EditModal.js`：添加密钥类型只读字段显示
   - `TableHead.js`：添加"密钥类型"表头
   - `TableRow.js`：添加密钥类型单元格，使用彩色标签区分

3. **air 主题** (`web/air/`)
   - `EditToken.js`：添加密钥类型 Select 组件（创建时可选，编辑时禁用）
   - `TokensTable.js`：添加"密钥类型"列，使用 Tag 组件显示

## 密钥格式说明

### OpenAI 格式（默认）
- 长度：48 字符
- 格式：纯随机字符串
- 示例：`abc123XYZ...`（48位）

### CC 格式
- 长度：约 108 字符
- 格式：`sk-ant-api03-` + 95位随机字符
- 示例：`sk-ant-api03-abc123XYZ...`（108位）

## 使用说明

### 创建密钥
1. 进入"令牌管理"页面
2. 点击"新建令牌"
3. 在"密钥类型"下拉框中选择：
   - **OpenAI 兼容格式**：适用于标准 OpenAI API
   - **Claude Code 格式 (sk-ant-)**：适用于 Claude Code 客户端
4. 填写其他信息后提交

### 验证密钥
- OpenAI 格式：使用原有的验证逻辑
- CC 格式：自动识别 `sk-ant-` 前缀并正确验证

## 数据库迁移
GORM 会自动处理数据库迁移：
- 添加 `key_type` 列（默认值 'openai'）
- 修改 `key` 列类型为 `varchar(150)`
- 已有数据自动设置为 'openai' 类型

## 兼容性
- ✅ 向后兼容：已有密钥默认为 OpenAI 格式，无需改动
- ✅ 密钥类型创建后不可修改（前端禁用编辑）
- ✅ 支持混合使用两种格式的密钥

## 编译和部署
```bash
# 编译后端
cd /workspaces/one-api
go build -o one-api

# 编译前端（需要分别编译三个主题）
cd web/default && npm run build
cd web/berry && npm run build
cd web/air && npm run build
```

## 文件清单
### 后端
- `model/token.go` - Token 模型定义
- `common/random/main.go` - 密钥生成函数
- `controller/token.go` - Token 控制器
- `middleware/auth.go` - 认证中间件

### 前端
- `web/default/src/pages/Token/EditToken.js`
- `web/default/src/components/TokensTable.js`
- `web/berry/src/views/Token/component/EditModal.js`
- `web/berry/src/views/Token/component/TableHead.js`
- `web/berry/src/views/Token/component/TableRow.js`
- `web/air/src/pages/Token/EditToken.js`
- `web/air/src/components/TokensTable.js`

## 测试建议
1. 创建 OpenAI 格式密钥并验证
2. 创建 CC 格式密钥并验证
3. 测试密钥在列表页面的显示
4. 测试编辑时密钥类型字段为禁用状态
5. 测试 API 调用时两种格式密钥都能正确验证
