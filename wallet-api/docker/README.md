# Wallet API Docker 部署文档

## 📋 目录结构

```
docker/
├── Dockerfile              # Docker 镜像构建文件
├── docker-compose.yml      # Docker Compose 编排文件
├── deploy.sh              # 一键部署脚本
├── .env.example           # 环境变量示例
├── init.sql               # 数据库初始化脚本
└── README.md              # 本文档
```

## 🚀 快速开始

### 方式 1：完整部署（包含 MySQL 和 Redis）

适合全新部署，会自动创建 MySQL 和 Redis 容器。

```bash
cd wallet-api/docker

# 1. 复制环境变量配置文件
cp .env.example .env

# 2. 编辑 .env 文件，修改配置
nano .env

# 3. 运行部署脚本
./deploy.sh
# 选择 1) Build and start (full deployment)
```

### 方式 2：连接已有数据库

如果你已经有运行中的 one-api 和 Redis，只需部署 wallet-api 服务。

```bash
cd wallet-api/docker

# 1. 编辑 .env 文件
cat > .env << 'EOF'
# 连接到已有的 one-api 数据库
SQL_DSN=root:your_password@tcp(your_host:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local

# 连接到已有的 Redis
REDIS_CONN_STRING=redis://your_redis_host:6379/0
REDIS_PASSWORD=your_redis_password

# JWT 密钥（生成随机密钥）
JWT_SECRET=$(openssl rand -hex 32)

# 服务端口
WALLET_API_PORT=8081
EOF

# 2. 只构建和启动 wallet-api 服务
docker-compose up -d wallet-api
```

## ⚙️ 环境变量配置

### 必需配置

| 变量 | 说明 | 示例 |
|------|------|------|
| `SQL_DSN` | 数据库连接字符串（必须与 one-api 共享） | `root:password@tcp(mysql:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local` |
| `REDIS_CONN_STRING` | Redis 连接字符串 | `redis://redis:6379/0` |
| `JWT_SECRET` | JWT 签名密钥（请修改为随机字符串） | 使用 `openssl rand -hex 32` 生成 |

### 可选配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `LOG_SQL_DSN` | 日志数据库连接（可选，不设置则使用主库） | - |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `WALLET_API_PORT` | 服务端口 | `8081` |
| `REDIS_STREAM_KEY` | Redis Stream 键名 | `wallet:balance:updates` |
| `REDIS_STREAM_MAXLEN` | Stream 最大消息数 | `10000` |

### MySQL 配置（仅用于 docker-compose 创建 MySQL）

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYSQL_ROOT_PASSWORD` | MySQL root 密码 | `password` |
| `MYSQL_DATABASE` | 数据库名 | `oneapi` |
| `MYSQL_USER` | 普通用户名（可选） | - |
| `MYSQL_PASSWORD` | 普通用户密码（可选） | - |

## 📦 Docker 镜像说明

### 多阶段构建

使用多阶段构建减小镜像体积：

1. **构建阶段**：使用 `golang:1.21-alpine` 编译
2. **运行阶段**：使用 `alpine:latest` 运行

### 镜像特性

- ✅ 支持 SQLite、MySQL、PostgreSQL
- ✅ 启用 CGO 以支持 SQLite
- ✅ 非 root 用户运行（安全）
- ✅ 健康检查配置
- ✅ 时区设置为 Asia/Shanghai
- ✅ 镜像大小约 30MB

## 🔧 部署脚本使用

`deploy.sh` 提供交互式部署管理：

```bash
./deploy.sh

# 选项：
# 1) Build and start (full deployment)  - 构建并启动所有服务
# 2) Build only                          - 仅构建镜像
# 3) Start services                      - 启动服务
# 4) Stop services                       - 停止服务
# 5) View logs                           - 查看日志
# 6) Restart services                    - 重启服务
# 7) Clean up                            - 清理容器和数据卷
```

## 🐳 Docker Compose 命令

### 基本操作

```bash
cd wallet-api/docker

# 启动服务（后台运行）
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f wallet-api

# 停止服务
docker-compose down

# 重启服务
docker-compose restart wallet-api

# 重新构建并启动
docker-compose up -d --build
```

### 进入容器

```bash
# 进入 wallet-api 容器
docker-compose exec wallet-api sh

# 进入 MySQL 容器
docker-compose exec mysql mysql -uroot -p

# 进入 Redis 容器
docker-compose exec redis redis-cli
```

### 查看健康状态

```bash
# 查看所有服务健康状态
docker-compose ps

# 测试 wallet-api 健康检查
curl http://localhost:8081/health
```

## 🔍 故障排查

### 检查服务状态

```bash
# 查看所有容器状态
docker-compose ps

# 查看 wallet-api 日志
docker-compose logs wallet-api

# 查看实时日志
docker-compose logs -f --tail=100 wallet-api
```

### 常见问题

#### 1. 数据库连接失败

```bash
# 检查 MySQL 是否启动
docker-compose ps mysql

# 查看 MySQL 日志
docker-compose logs mysql

# 测试数据库连接
docker-compose exec mysql mysql -uroot -ppassword -e "SHOW DATABASES;"
```

#### 2. Redis 连接失败

```bash
# 检查 Redis 是否启动
docker-compose ps redis

# 测试 Redis 连接
docker-compose exec redis redis-cli ping
```

#### 3. 服务无法启动

```bash
# 查看详细错误日志
docker-compose logs wallet-api

# 重新构建镜像
docker-compose build --no-cache wallet-api

# 清理并重启
docker-compose down -v
docker-compose up -d
```

#### 4. 端口被占用

```bash
# 检查端口占用
netstat -tuln | grep 8081

# 修改 .env 中的 WALLET_API_PORT
echo "WALLET_API_PORT=8082" >> .env

# 重启服务
docker-compose down
docker-compose up -d
```

## 🔐 安全建议

### 生产环境部署

1. **修改默认密码**
```bash
# 生成随机 JWT 密钥
openssl rand -hex 32

# 修改 MySQL 密码
MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32)
```

2. **使用 Docker Secrets**（Swarm 模式）
```yaml
secrets:
  jwt_secret:
    external: true
  mysql_password:
    external: true
```

3. **限制网络访问**
```yaml
# 仅暴露必要端口
ports:
  - "127.0.0.1:8081:8081"  # 仅本地访问
```

4. **使用反向代理**
```bash
# 推荐使用 Nginx 或 Traefik 作为反向代理
# 配置 HTTPS 和访问控制
```

## 📊 监控和日志

### 日志管理

```bash
# 查看最近 100 行日志
docker-compose logs --tail=100 wallet-api

# 持续查看日志
docker-compose logs -f wallet-api

# 查看错误日志
docker-compose logs wallet-api 2>&1 | grep -i error
```

### 资源监控

```bash
# 查看容器资源使用
docker stats wallet-api

# 查看详细信息
docker inspect wallet-api
```

### 备份数据

```bash
# 备份 MySQL 数据
docker-compose exec mysql mysqldump -uroot -ppassword oneapi > backup.sql

# 备份 Redis 数据
docker-compose exec redis redis-cli SAVE
docker cp wallet-redis:/data/dump.rdb ./redis-backup.rdb
```

## 🌐 连接已有 one-api

如果你已经有运行中的 one-api，可以这样连接：

### 1. 确认 one-api 的网络

```bash
# 查看 one-api 的网络
docker network ls
docker inspect <one-api-network-name>
```

### 2. 修改 docker-compose.yml

```yaml
services:
  wallet-api:
    # ... 其他配置
    networks:
      - wallet-network
      - one-api-network  # 添加 one-api 的网络

networks:
  wallet-network:
    driver: bridge
  one-api-network:
    external: true  # 使用外部已存在的网络
```

### 3. 配置环境变量

```bash
# 使用 one-api 的数据库和 Redis
SQL_DSN=root:password@tcp(one-api-mysql:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local
REDIS_CONN_STRING=redis://one-api-redis:6379/0
```

## 🎯 API 测试

服务启动后，测试 API：

```bash
# 健康检查
curl http://localhost:8081/health

# 用户认证
curl -X POST http://localhost:8081/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"username":"root","password":"123456"}'

# 查询余额（需要替换 TOKEN）
curl -X GET http://localhost:8081/api/v1/balance \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 📈 性能优化

### 1. 调整资源限制

```yaml
services:
  wallet-api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 256M
```

### 2. 数据库连接池

已在代码中配置：
- MaxIdleConns: 10
- MaxOpenConns: 100
- ConnMaxLifetime: 1 hour

### 3. Redis 优化

```bash
# 在 .env 中添加
REDIS_MAXMEMORY=256mb
REDIS_MAXMEMORY_POLICY=allkeys-lru
```

## 🔄 更新部署

```bash
# 1. 拉取最新代码
cd wallet-api
git pull

# 2. 重新构建并部署
cd docker
./deploy.sh
# 选择 1) Build and start
```

## 📞 支持

- 查看完整 API 文档：`../README.md`
- 消费者示例代码：`../examples/consumer-example.md`
- 问题反馈：提交 Issue

---

**部署前请务必：**
1. ✅ 修改 JWT_SECRET 为随机字符串
2. ✅ 修改 MySQL 默认密码
3. ✅ 确认数据库连接信息正确
4. ✅ 备份现有数据库
