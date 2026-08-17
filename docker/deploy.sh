#!/bin/bash

# One-API Codex 一键部署脚本

set -e

echo "=========================================="
echo "  One-API Codex 版本部署脚本"
echo "=========================================="
echo ""

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "错误: 未检测到 Docker，请先安装 Docker"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "错误: 未检测到 Docker Compose，请先安装 Docker Compose"
    exit 1
fi

# 检查镜像文件是否存在
if [ ! -f "one-api-codex.tar" ]; then
    echo "错误: 找不到镜像文件 one-api-codex.tar"
    exit 1
fi

echo "步骤 1/4: 加载 Docker 镜像..."
docker load -i one-api-codex.tar

echo ""
echo "步骤 2/4: 验证镜像..."
if docker images | grep -q "one-api-codex"; then
    echo "✓ 镜像加载成功"
else
    echo "✗ 镜像加载失败"
    exit 1
fi

echo ""
echo "步骤 3/4: 创建数据目录..."
mkdir -p data/oneapi
mkdir -p data/mysql
mkdir -p logs

echo ""
echo "步骤 4/4: 启动服务..."
docker-compose up -d

echo ""
echo "=========================================="
echo "  部署完成！"
echo "=========================================="
echo ""
echo "服务访问地址: http://localhost:3000"
echo "默认用户名: root"
echo "默认密码: 123456"
echo ""
echo "请使用以下命令查看日志:"
echo "  docker-compose logs -f one-api"
echo ""
echo "重要提示:"
echo "  1. 首次登录后请立即修改密码"
echo "  2. 创建 Codex 类型密钥以使用新功能"
echo "  3. 详细文档请查看 CODEX_FEATURE.md"
echo ""
echo "常用命令:"
echo "  启动服务: docker-compose up -d"
echo "  停止服务: docker-compose down"
echo "  查看状态: docker-compose ps"
echo "  查看日志: docker-compose logs -f"
echo ""
