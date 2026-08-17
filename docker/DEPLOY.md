# One-API Codex 版本部署指南

## 📦 文件说明

本目录包含以下文件：

- `one-api-codex.tar` - Docker 镜像文件（约 26MB）
- `docker-compose.yml` - Docker Compose 配置文件
- `.env.example` - 环境变量示例文件
- `CODEX_FEATURE.md` - Codex 功能使用文档
- `IMPLEMENTATION_SUMMARY.md` - 技术实现总结
- `DEPLOY.md` - 本部署文档

## 🚀 快速部署

### 1. 上传文件到服务器

将整个 `docker` 文件夹上传到服务器，例如：

```bash
scp -r ./docker user@your-server:/opt/one-api-codex/
```

### 2. 加载 Docker 镜像

```bash
cd /opt/one-api-codex
docker load -i one-api-codex.tar
```

验证镜像已加载：
```bash
docker images | grep one-api-codex
```

应该看到：
```
one-api-codex    latest    <IMAGE_ID>    <SIZE>
```

### 3. 配置环境变量（可选）

如果需要自定义配置，复制并编辑环境变量文件：

```bash
cp .env.example .env
nano .env
```

### 4. 启动服务

```bash
docker-compose up -d
```

查看日志：
```bash
docker-compose logs -f one-api
```

### 5. 访问服务

打开浏览器访问：
```
http://your-server-ip:3000
```

默认管理员账号：
- 用户名: `root`
- 密码: `123456`

**重要：首次登录后请立即修改密码！**

## 🔧 配置说明

### 数据库选项

#### 选项 1：使用 MySQL（推荐生产环境）

默认配置已启用 MySQL，会自动创建容器。数据持久化在 `./data/mysql/` 目录。

#### 选项 2：使用 SQLite（适合测试环境）

编辑 `docker-compose.yml`，注释掉 MySQL 相关配置：

```yaml
services:
  one-api:
    environment:
      # - SQL_DSN=oneapi:123456@tcp(db:3306)/one-api  # 注释掉这行
    # depends_on:                                      # 注释掉依赖
    #   - db
    #   - redis

  # db:  # 注释掉整个 db 服务
  #   ...
```

SQLite 数据库文件会保存在 `./data/oneapi/one-api.db`。

### Redis 配置

默认使用 Redis 容器，如需使用外部 Redis：

```yaml
environment:
  - REDIS_CONN_STRING=redis://your-redis-host:6379
```

### 端口配置

修改 `docker-compose.yml` 中的端口映射：

```yaml
ports:
  - "8080:3000"  # 改为你需要的端口
```

## 📝 使用 Codex 功能

### 1. 创建 Codex 类型密钥

1. 登录管理后台
2. 进入 "令牌" 页面
3. 点击 "新建令牌"
4. 在 "密钥类型" 下拉框中选择 **"Codex 格式（自动转换 /v1/responses）"**
5. 填写其他信息并保存

### 2. 使用密钥

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
        "content": [
          {
            "type": "text",
            "text": "Hello, how are you?"
          }
        ]
      }
    ],
    "stream": true,
    "max_tokens": 1000
  }'
```

系统会自动：
- 将请求转换为 Chat Completions 格式
- 发送到配置的渠道
- 将响应转换回 Responses API 格式
- 返回标准的 SSE 事件流

### 3. 响应格式

你会收到以下 SSE 事件：

```
event: response.output_item.added
data: {...}

event: response.output_item.delta
data: {"type":"response.output_item.delta","delta":{"text":"Hello"}}

event: response.output_item.done
data: {...}

event: response.done
data: {"type":"response.done","status":"completed"}

data: [DONE]
```

详细使用说明请参考 `CODEX_FEATURE.md`。

## 🔄 更新服务

```bash
cd /opt/one-api-codex
docker-compose down
docker load -i one-api-codex-new.tar  # 加载新镜像
docker-compose up -d
```

## 🛠️ 常用命令

### 查看运行状态
```bash
docker-compose ps
```

### 查看日志
```bash
docker-compose logs -f one-api
docker-compose logs -f redis
docker-compose logs -f db
```

### 重启服务
```bash
docker-compose restart one-api
```

### 停止服务
```bash
docker-compose down
```

### 备份数据

#### MySQL 数据库备份
```bash
docker exec mysql mysqldump -u oneapi -p123456 one-api > backup.sql
```

#### SQLite 数据库备份
```bash
cp ./data/oneapi/one-api.db ./backup/one-api-$(date +%Y%m%d).db
```

### 恢复数据

#### MySQL 恢复
```bash
docker exec -i mysql mysql -u oneapi -p123456 one-api < backup.sql
```

#### SQLite 恢复
```bash
cp ./backup/one-api-20240816.db ./data/oneapi/one-api.db
```

## 🐛 故障排查

### 服务无法启动

1. 检查端口是否被占用：
```bash
netstat -tuln | grep 3000
```

2. 查看详细日志：
```bash
docker-compose logs one-api
```

3. 检查镜像是否正确加载：
```bash
docker images | grep one-api-codex
```

### 数据库连接失败

1. 检查 MySQL 容器状态：
```bash
docker-compose ps db
docker-compose logs db
```

2. 验证数据库连接：
```bash
docker exec -it mysql mysql -u oneapi -p123456 -e "SHOW DATABASES;"
```

### Codex 功能不工作

1. 确认密钥类型为 "codex"
2. 查看日志中的转换信息：
```bash
docker-compose logs one-api | grep -i codex
```

3. 验证请求格式是否正确（参考 CODEX_FEATURE.md）

## 📊 监控与维护

### 磁盘空间监控

定期检查数据目录大小：
```bash
du -sh ./data/
```

### 日志轮转

建议配置日志轮转以防止日志文件过大。编辑 `/etc/logrotate.d/one-api`：

```
/opt/one-api-codex/logs/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

### 性能优化

1. **数据库优化**：定期清理过期日志
2. **Redis 内存**：根据需要调整 Redis 配置
3. **连接池**：根据负载调整数据库连接池大小

## 🔒 安全建议

1. **修改默认密码**：首次登录后立即修改 root 密码
2. **数据库密码**：修改 `docker-compose.yml` 中的数据库密码
3. **SESSION_SECRET**：设置强随机字符串
4. **防火墙**：配置防火墙规则，只开放必要端口
5. **HTTPS**：生产环境建议使用 Nginx 反向代理配置 SSL
6. **定期备份**：设置自动备份任务

## 📞 支持

如有问题：
1. 查看 `CODEX_FEATURE.md` 了解功能详情
2. 查看 `IMPLEMENTATION_SUMMARY.md` 了解技术细节
3. 检查日志文件获取错误信息

## 🎯 版本信息

- **镜像名称**: one-api-codex:latest
- **基础版本**: One-API (最新版)
- **新增功能**: Codex 密钥类型 - 自动转换 /v1/responses 到 /v1/chat/completions
- **构建日期**: 2024-08-16

## 📄 许可证

本项目基于 One-API 项目，遵循其原有许可证。
