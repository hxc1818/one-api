# One-API Docker 部署包

本目录包含 One-API 的完整 Docker 部署方案和数据库迁移工具。

## 📁 文件说明

```
docker/
├── Dockerfile              # Docker 镜像构建文件
├── docker-compose.yml      # Docker Compose 配置文件
├── .env.example            # 环境变量配置示例
├── build.sh               # 镜像构建脚本
├── migrate.sh             # 数据库迁移向导（交互式）
├── migrate_mysql.sql      # MySQL 数据库迁移 SQL
├── migrate_postgres.sql   # PostgreSQL 数据库迁移 SQL
├── migrate_sqlite.sql     # SQLite 数据库迁移 SQL
├── MIGRATION_GUIDE.md     # 详细迁移手册
└── README.md             # 本文件
```

## 🚀 快速开始

### 新部署

如果您是首次部署 One-API：

```bash
# 1. 构建镜像
./build.sh

# 2. 准备配置
cp .env.example .env
# 编辑 .env 文件，根据需要修改配置

# 3. 启动服务
docker-compose up -d

# 4. 查看日志
docker-compose logs -f one-api

# 5. 访问系统
# 浏览器打开: http://localhost:3000
# 默认账号: root
# 默认密码: 123456
```

### 从旧版本迁移

如果您已有旧版本的 One-API 正在运行：

```bash
# 1. 停止旧服务（但不要删除数据！）
docker-compose down

# 2. 备份数据
# 详见 MIGRATION_GUIDE.md 的"数据备份"章节

# 3. 执行数据库迁移
./migrate.sh

# 4. 构建新镜像
./build.sh

# 5. 更新配置文件
cp .env.example .env
# 将旧配置迁移到新 .env 文件

# 6. 启动新服务
docker-compose up -d

# 7. 验证功能
# 详见 MIGRATION_GUIDE.md 的"验证与测试"章节
```

## 📋 本次更新内容

### 新增功能

1. **用户速率限制**
   - 支持设置每分钟(RPM)、每天(RPD)、每周(RPW)请求数限制
   - 超出限制返回 429 状态码
   - 需要 Redis 支持（强烈推荐）

2. **兑换码批量管理**
   - 支持按名称批量禁用兑换码
   - 在兑换管理页面一键操作

3. **模型广场功能**
   - 在设置中配置模型广场链接
   - 导航栏自动显示入口按钮
   - 新标签页打开

4. **令牌已用额度修复**
   - 修复令牌使用量统计显示问题
   - 新增 `used_quota` 字段追踪

### 数据库变更

#### users 表新增字段
- `rpm` - 每分钟请求数限制（INT，默认 0）
- `rpd` - 每天请求数限制（INT，默认 0）
- `rpw` - 每周请求数限制（INT，默认 0）

#### tokens 表新增字段
- `used_quota` - 已使用配额（BIGINT，默认 0）

## ⚙️ 配置说明

### 数据库选择

One-API 支持三种数据库：

#### SQLite（默认）
适合小规模部署（< 100 并发用户）

```bash
SQL_DSN=one-api.db
```

#### MySQL/MariaDB
适合中大规模部署

```bash
SQL_DSN=root:your_password@tcp(mysql:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local
```

#### PostgreSQL
适合大规模企业部署

```bash
SQL_DSN=postgres://user:your_password@postgres:5432/oneapi?sslmode=disable
```

### Redis 配置（强烈推荐）

Redis 用于：
- 用户速率限制
- 数据缓存
- 分布式锁

```bash
REDIS_CONN_STRING=redis://redis:6379
```

### 性能优化

启用批量更新以减少数据库压力：

```bash
BATCH_UPDATE_ENABLED=true
BATCH_UPDATE_INTERVAL=5
```

## 🔧 常用命令

### 服务管理

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f one-api
```

### 数据库操作

```bash
# 进入 SQLite 数据库
docker exec -it one-api sqlite3 /data/one-api.db

# 进入 MySQL 数据库
docker exec -it mysql mysql -u root -p oneapi

# 进入 PostgreSQL 数据库
docker exec -it postgres psql -U user -d oneapi
```

### 备份与恢复

```bash
# 备份 SQLite
docker cp one-api:/data/one-api.db ./backup/

# 恢复 SQLite
docker cp ./backup/one-api.db one-api:/data/

# 备份 MySQL
docker exec mysql mysqldump -u root -p oneapi > backup.sql

# 恢复 MySQL
docker exec -i mysql mysql -u root -p oneapi < backup.sql
```

## 📊 资源要求

### 最小配置
- CPU: 1 核心
- 内存: 512 MB
- 磁盘: 5 GB

### 推荐配置
- CPU: 2 核心
- 内存: 2 GB
- 磁盘: 20 GB（SSD）
- Redis: 256 MB

### 生产环境
- CPU: 4+ 核心
- 内存: 4+ GB
- 磁盘: 50+ GB（SSD）
- Redis: 1+ GB
- MySQL/PostgreSQL 独立部署

## 🔐 安全建议

1. **修改默认密码**
   - 首次登录后立即修改 root 密码

2. **使用强密码**
   - 数据库密码至少 16 位
   - SESSION_SECRET 使用随机字符串

3. **启用 HTTPS**
   - 使用 Nginx/Caddy 反向代理
   - 配置 SSL 证书

4. **限制访问**
   - 使用防火墙规则
   - 仅开放必要端口

5. **定期备份**
   - 每天自动备份数据库
   - 保留至少 7 天备份

6. **更新维护**
   - 定期更新到最新版本
   - 关注安全公告

## 📚 文档链接

- **详细迁移指南**: [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md)
- **功能变更说明**: [../IMPLEMENTATION_CHANGES.md](../IMPLEMENTATION_CHANGES.md)
- **官方文档**: https://github.com/songquanpeng/one-api

## ❓ 故障排查

### 无法启动

```bash
# 查看日志
docker-compose logs one-api

# 检查端口占用
netstat -tlnp | grep 3000

# 检查数据卷权限
ls -la ./data/
```

### 数据库连接失败

```bash
# 检查数据库容器
docker-compose ps

# 测试连接
docker-compose exec one-api ping mysql

# 查看环境变量
docker-compose exec one-api env | grep SQL
```

### Redis 连接失败

```bash
# 检查 Redis
docker-compose exec redis redis-cli ping

# 查看网络
docker network inspect one-api-network
```

更多问题请参考 [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) 的"常见问题"章节。

## 🆘 获取帮助

如果遇到问题：

1. 查看本目录的 `MIGRATION_GUIDE.md`
2. 查看根目录的 `IMPLEMENTATION_CHANGES.md`
3. 搜索 GitHub Issues
4. 提交新的 Issue（附带日志）

## 📝 许可证

本项目基于原 One-API 项目，遵循相同的开源协议。

---

**祝您部署顺利！**
