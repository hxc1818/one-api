# 🚀 One API 业务服务器部署清单

## 📦 方案一：最简部署（推荐）

### 需要上传的文件
```
one-api-latest.tar.gz    # 26 MB - Docker 镜像文件
```

### 服务器操作步骤

1. **上传镜像文件**
```bash
# 在本地执行
scp one-api-latest.tar.gz user@your-server:/path/to/upload/
```

2. **在服务器上导入镜像**
```bash
# SSH 登录服务器
ssh user@your-server

# 导入镜像
docker load < one-api-latest.tar.gz

# 验证镜像
docker images | grep one-api
```

3. **启动服务**
```bash
# 创建数据目录
mkdir -p /opt/one-api/data

# 启动容器
docker run -d \
  --name one-api \
  --restart always \
  -p 3000:3000 \
  -v /opt/one-api/data:/data \
  -e TZ=Asia/Shanghai \
  -e SESSION_SECRET=$(openssl rand -hex 16) \
  one-api:latest

# 查看日志
docker logs -f one-api
```

---

## 📦 方案二：完整部署（带 MySQL 和 Redis）

### 需要上传的文件
```
one-api-latest.tar.gz       # 26 MB - Docker 镜像文件
docker-compose.yml          # Docker Compose 配置
start.sh                    # 启动脚本（可选）
DOCKER_DEPLOYMENT.md        # 部署文档（可选）
```

### 服务器操作步骤

1. **上传文件**
```bash
# 在本地执行
scp one-api-latest.tar.gz docker-compose.yml start.sh user@your-server:/opt/one-api/
```

2. **在服务器上操作**
```bash
# SSH 登录服务器
ssh user@your-server
cd /opt/one-api

# 导入镜像
docker load < one-api-latest.tar.gz

# 修改 docker-compose.yml 中的镜像名（如果需要）
# 将 image: "${REGISTRY:-docker.io}/justsong/one-api:latest" 
# 改为 image: "one-api:latest"

# 启动服务
chmod +x start.sh
./start.sh compose

# 或直接使用 docker-compose
docker-compose up -d
```

---

## 📦 方案三：服务器上构建（不推荐，耗时长）

### 需要上传的文件
```
整个项目目录（约 20 MB）
```

### 服务器操作步骤
```bash
# 上传整个项目
scp -r /workspaces/one-api user@your-server:/opt/

# 在服务器上构建
ssh user@your-server
cd /opt/one-api
docker build -t one-api:latest -f Dockerfile .
```

---

## ✅ 推荐方案对比

| 方案 | 上传大小 | 启动速度 | 优点 | 缺点 |
|------|---------|---------|------|------|
| **方案一** | 26 MB | ⚡ 最快 | 简单快速，即导即用 | 需要手动配置环境变量 |
| **方案二** | 26 MB + 配置 | ⚡ 快 | 完整功能，一键部署 | 需要更多资源（MySQL+Redis） |
| **方案三** | 20 MB | 🐌 慢（5分钟+） | 可自定义构建 | 耗时长，需要服务器资源 |

**👍 建议：使用方案一或方案二**

---

## 🔍 服务器环境检查

### 在服务器上执行检查
```bash
# 检查 Docker
docker --version
# 需要 Docker 20.10+

# 检查 Docker Compose（如果使用方案二）
docker-compose --version
# 或
docker compose version

# 检查端口占用
netstat -tunlp | grep 3000
# 确保 3000 端口未被占用

# 检查磁盘空间
df -h
# 确保至少有 2GB 可用空间
```

---

## 📋 部署后验证

```bash
# 1. 检查容器状态
docker ps | grep one-api

# 2. 检查日志
docker logs one-api --tail 50

# 3. 测试访问
curl http://localhost:3000/api/status

# 4. 浏览器访问
# http://your-server-ip:3000
```

---

## 🔐 安全建议

1. **修改默认密码**
   - 登录后立即修改 root 账号密码

2. **配置防火墙**
```bash
# 如果使用 firewalld
firewall-cmd --permanent --add-port=3000/tcp
firewall-cmd --reload

# 如果使用 ufw
ufw allow 3000/tcp
```

3. **配置 Nginx 反向代理**（推荐）
```bash
# 使用 HTTPS 和域名访问
# 参考 DOCKER_DEPLOYMENT.md 中的 Nginx 配置
```

4. **定期备份数据**
```bash
# 备份数据目录
tar -czf one-api-backup-$(date +%Y%m%d).tar.gz /opt/one-api/data
```

---

## 🆘 故障排查

### 容器无法启动
```bash
# 查看详细日志
docker logs one-api

# 检查端口占用
netstat -tunlp | grep 3000

# 重启容器
docker restart one-api
```

### 无法访问服务
```bash
# 检查容器状态
docker ps -a | grep one-api

# 检查防火墙
iptables -L -n | grep 3000

# 检查服务是否正常
docker exec one-api ps aux
```

---

## 📞 快速参考

- **访问地址**: http://服务器IP:3000
- **默认用户名**: root
- **默认密码**: 123456
- **数据目录**: /opt/one-api/data （或你指定的目录）
- **日志查看**: `docker logs -f one-api`
- **重启服务**: `docker restart one-api`

---

**部署完成后，建议阅读 DOCKER_DEPLOYMENT.md 了解更多配置选项！**
