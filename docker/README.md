# One-API Codex 版本

## 快速开始

### 方法 1: 使用一键部署脚本（推荐）

```bash
chmod +x deploy.sh
./deploy.sh
```

### 方法 2: 手动部署

```bash
# 1. 加载镜像
docker load -i one-api-codex.tar

# 2. 启动服务
docker-compose up -d

# 3. 查看日志
docker-compose logs -f one-api
```

## 访问服务

服务启动后，访问：`http://your-server-ip:3000`

默认账号：
- 用户名: `root`
- 密码: `123456`

## 新增功能：Codex 密钥类型

本版本新增了 **Codex 格式** 密钥类型，可以自动将 `/v1/responses` 请求转换为 `/v1/chat/completions` 请求。

### 使用方法

1. 登录后台，进入"令牌"页面
2. 创建新令牌，选择"Codex 格式（自动转换 /v1/responses）"
3. 使用该密钥请求 `/v1/responses` 端点即可自动转换

### 示例请求

```bash
curl -N http://your-server:3000/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-codex-key" \
  -d '{
    "model": "gpt-4",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [{"type": "text", "text": "Hello"}]
      }
    ],
    "stream": true
  }'
```

## 文档

- `DEPLOY.md` - 详细部署文档
- `CODEX_FEATURE.md` - Codex 功能详细说明
- `IMPLEMENTATION_SUMMARY.md` - 技术实现总结

## 文件列表

```
docker/
├── one-api-codex.tar           # Docker 镜像（26MB）
├── docker-compose.yml          # Docker Compose 配置
├── deploy.sh                   # 一键部署脚本
├── .env.example               # 环境变量示例
├── README.md                  # 本文件
├── DEPLOY.md                  # 详细部署文档
├── CODEX_FEATURE.md          # 功能说明文档
└── IMPLEMENTATION_SUMMARY.md  # 技术实现文档
```

## 系统要求

- Docker 20.10+
- Docker Compose 1.29+
- 至少 1GB 可用内存
- 至少 2GB 可用磁盘空间

## 常见问题

### Q: 端口 3000 被占用怎么办？
A: 编辑 `docker-compose.yml`，修改端口映射，例如改为 `8080:3000`

### Q: 如何使用 SQLite 而不是 MySQL？
A: 编辑 `docker-compose.yml`，注释掉 MySQL 相关配置，详见 DEPLOY.md

### Q: Codex 功能如何工作？
A: 详细说明请查看 `CODEX_FEATURE.md`

### Q: 如何备份数据？
A: 
- MySQL: `docker exec mysql mysqldump -u oneapi -p123456 one-api > backup.sql`
- SQLite: `cp ./data/oneapi/one-api.db ./backup/`

## 技术支持

如遇问题：
1. 查看 `docker-compose logs one-api` 日志
2. 参考 `DEPLOY.md` 故障排查章节
3. 检查 `IMPLEMENTATION_SUMMARY.md` 了解技术细节

## 更新日志

### v1.0.0 (2024-08-16)
- ✨ 新增 Codex 密钥类型
- ✨ 自动转换 /v1/responses 到 /v1/chat/completions
- ✨ 支持流式响应格式转换
- ✨ 正确的计费逻辑
- 📝 完善的文档

## 许可证

基于 One-API 项目开发，遵循其原有许可证。
