# One-API Codex 版本 - 文件清单

## 📦 打包内容

本 docker 文件夹包含了部署所需的全部文件：

### 核心文件
- ✅ `one-api-codex.tar` (26MB) - Docker 镜像文件
- ✅ `docker-compose.yml` - Docker Compose 配置文件
- ✅ `.env.example` - 环境变量示例

### 部署脚本
- ✅ `deploy.sh` - 一键部署脚本（可执行）
- ✅ `test.sh` - 功能测试脚本（可执行）

### 文档文件
- ✅ `README.md` - 快速开始指南
- ✅ `DEPLOY.md` - 详细部署文档
- ✅ `CODEX_FEATURE.md` - Codex 功能使用说明
- ✅ `IMPLEMENTATION_SUMMARY.md` - 技术实现总结
- ✅ `FILELIST.md` - 本文件清单

## 📋 文件说明

### one-api-codex.tar
Docker 镜像文件，包含：
- 编译好的 Go 后端程序
- 构建好的前端静态文件（3个主题）
- 所有依赖库
- Alpine Linux 基础镜像

### docker-compose.yml
定义了三个服务：
- `one-api` - 主应用服务
- `redis` - Redis 缓存服务
- `db` - MySQL 数据库服务（可选，可改用 SQLite）

### deploy.sh
自动化部署脚本，执行：
1. 检查 Docker 环境
2. 加载镜像
3. 创建数据目录
4. 启动服务

### test.sh
功能测试脚本，用于验证：
1. Codex 转换功能是否正常
2. 普通 API 请求不受影响

## 🚀 使用流程

### 第一步：上传到服务器
```bash
# 将整个 docker 文件夹上传到服务器
scp -r ./docker user@server:/opt/one-api-codex/
```

### 第二步：部署
```bash
# SSH 到服务器
ssh user@server

# 进入目录
cd /opt/one-api-codex

# 一键部署
chmod +x deploy.sh
./deploy.sh
```

### 第三步：访问
浏览器打开：`http://server-ip:3000`
- 用户名: root
- 密码: 123456

### 第四步：创建 Codex 密钥
1. 登录后台
2. 进入"令牌"页面
3. 创建新令牌，选择"Codex 格式"

### 第五步：测试（可选）
```bash
chmod +x test.sh
./test.sh http://localhost:3000 sk-your-key
```

## 📊 文件大小

```
one-api-codex.tar        ~26MB   (Docker 镜像)
其他所有文件              ~30KB   (文档和脚本)
总计                      ~26MB
```

## 🔧 自定义配置

所有配置都可以通过编辑 `docker-compose.yml` 修改：

- **端口**：修改 `ports: - "3000:3000"` 
- **数据库**：注释 MySQL 部分使用 SQLite
- **Redis**：使用外部 Redis 或注释掉
- **时区**：修改 `TZ` 环境变量
- **日志目录**：修改 `volumes` 挂载路径

详细说明请查看 `DEPLOY.md`。

## ✅ 检查清单

部署前确认：
- [ ] 服务器已安装 Docker 和 Docker Compose
- [ ] 端口 3000 未被占用（或已修改配置）
- [ ] 有足够的磁盘空间（至少 2GB）
- [ ] 有足够的内存（建议 1GB+）

部署后确认：
- [ ] 服务正常启动 (`docker-compose ps`)
- [ ] 可以访问 Web 界面
- [ ] 能够登录管理后台
- [ ] 已修改默认密码
- [ ] 创建了测试密钥
- [ ] Codex 功能测试通过

## 📝 版本信息

- **版本**: v1.0.0-codex
- **构建日期**: 2024-08-16
- **基础版本**: One-API (最新)
- **新增功能**: Codex 密钥类型 + /v1/responses 自动转换

## 🎯 新功能特性

✨ **Codex 密钥类型**
- 自动将 /v1/responses 请求转换为 /v1/chat/completions
- 支持流式响应格式转换
- 正确的计费逻辑
- 完全向后兼容

## 📞 技术支持

遇到问题？
1. 查看 `DEPLOY.md` 故障排查章节
2. 检查 `docker-compose logs one-api`
3. 参考 `CODEX_FEATURE.md` 使用说明
4. 查看 `IMPLEMENTATION_SUMMARY.md` 技术细节

## ⚠️ 重要提示

1. **修改密码**：首次登录后立即修改默认密码
2. **数据备份**：定期备份 `./data/` 目录
3. **安全配置**：生产环境建议配置 HTTPS
4. **防火墙**：只开放必要的端口

---

**准备就绪！所有文件已打包在 docker 文件夹中，可以直接部署到生产环境。**
