# One API Docker 部署指南

## 📦 镜像信息

- **镜像名称**: `one-api:latest`
- **镜像大小**: 110 MB
- **构建时间**: 2026-08-16
- **版本**: 2787c0e

## 🚀 快速启动

### 方式一：单容器运行（使用 SQLite）

```bash
docker run -d \
  --name one-api \
  --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -e SESSION_SECRET=random_string_please_change_me \
  -v $(pwd)/data:/data \
  one-api:latest
```

访问: http://localhost:3000

### 方式二：使用 Docker Compose（完整部署，包含 MySQL 和 Redis）

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f one-api

# 停止服务
docker-compose down

# 停止并删除数据
docker-compose down -v
```

## 🔧 环境变量配置

| 环境变量 | 说明 | 默认值 | 必填 |
|---------|------|--------|------|
| `PORT` | 服务端口 | 3000 | 否 |
| `SESSION_SECRET` | Session 密钥 | - | 是 |
| `SQL_DSN` | 数据库连接串 | SQLite | 否 |
| `REDIS_CONN_STRING` | Redis 连接串 | - | 否 |
| `TZ` | 时区 | UTC | 否 |
| `NODE_TYPE` | 节点类型（slave） | - | 否 |
| `SYNC_FREQUENCY` | 同步频率（秒） | - | 否 |
| `FRONTEND_BASE_URL` | 前端地址 | - | 否 |
| `CHANNEL_TEST_FREQUENCY` | 渠道测试频率 | - | 否 |
| `BATCH_UPDATE_ENABLED` | 批量更新开关 | false | 否 |

### 数据库连接示例

**MySQL**:
```
SQL_DSN=username:password@tcp(host:3306)/database
```

**PostgreSQL**:
```
SQL_DSN=host=localhost user=username password=password dbname=oneapi port=5432 sslmode=disable
```

## 📁 目录结构

```
.
├── data/               # 数据目录
│   ├── oneapi/        # One API 数据（SQLite 数据库等）
│   └── mysql/         # MySQL 数据（如果使用 docker-compose）
├── logs/              # 日志目录
└── docker-compose.yml # Docker Compose 配置
```

## 🔐 默认账号

- **用户名**: root
- **密码**: 123456

**⚠️ 首次登录后请立即修改密码！**

## 🛠️ 常用命令

### 查看容器状态
```bash
docker ps -a | grep one-api
```

### 查看日志
```bash
docker logs -f one-api
```

### 进入容器
```bash
docker exec -it one-api sh
```

### 重启容器
```bash
docker restart one-api
```

### 停止容器
```bash
docker stop one-api
```

### 删除容器
```bash
docker rm -f one-api
```

## 📊 数据备份

### SQLite 备份
```bash
# 备份数据库
docker exec one-api cp /data/one-api.db /data/one-api.db.backup

# 导出备份文件
docker cp one-api:/data/one-api.db.backup ./one-api-$(date +%Y%m%d).db
```

### MySQL 备份
```bash
docker exec mysql mysqldump -u root -p'OneAPI@justsong' one-api > one-api-$(date +%Y%m%d).sql
```

## 🔄 更新镜像

### 重新构建镜像
```bash
docker build -t one-api:latest -f Dockerfile .
```

### 使用新镜像重启
```bash
docker stop one-api
docker rm one-api
# 然后使用快速启动命令重新运行
```

## 🌐 Nginx 反向代理配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## ❓ 常见问题

### 1. 容器启动失败
```bash
# 检查日志
docker logs one-api

# 检查端口占用
netstat -tunlp | grep 3000
```

### 2. 数据库连接失败
- 检查 `SQL_DSN` 环境变量是否正确
- 确保数据库服务已启动
- 检查网络连接

### 3. 前端无法访问
- 确认容器已启动: `docker ps | grep one-api`
- 检查端口映射: `docker port one-api`
- 检查防火墙设置

## 📈 性能优化

### 启用内存缓存和 Redis
```bash
docker run -d \
  --name one-api \
  --restart always \
  -p 3000:3000 \
  -e REDIS_CONN_STRING=redis://your-redis-host:6379 \
  -v $(pwd)/data:/data \
  one-api:latest
```

### 多机部署
```bash
# 主节点
docker run -d --name one-api-master -p 3000:3000 ...

# 从节点
docker run -d \
  --name one-api-slave \
  -p 3001:3000 \
  -e NODE_TYPE=slave \
  -e SYNC_FREQUENCY=60 \
  -e FRONTEND_BASE_URL=https://your-master-domain.com \
  -e SQL_DSN=... \
  one-api:latest
```

## 📞 技术支持

- GitHub: https://github.com/songquanpeng/one-api
- 文档: 查看项目 README.md

---

构建时间: 2026-08-16 03:55:08 UTC
