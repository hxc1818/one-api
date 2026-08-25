#!/bin/bash

# =========================================
# One-API Docker 构建脚本
# =========================================

set -e

echo "======================================"
echo "One-API Docker 构建脚本"
echo "======================================"
echo ""

# 检查是否在项目根目录
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "错误：请在项目根目录运行此脚本"
    exit 1
fi

# 设置变量
IMAGE_NAME="one-api"
VERSION=$(cat VERSION 2>/dev/null || echo "latest")
DOCKERFILE="docker/Dockerfile"

echo "镜像名称: ${IMAGE_NAME}"
echo "版本号: ${VERSION}"
echo ""

# 询问用户是否构建多平台镜像
read -p "是否构建多平台镜像 (amd64 + arm64)? [y/N]: " BUILD_MULTI
BUILD_MULTI=${BUILD_MULTI:-n}

if [[ "$BUILD_MULTI" =~ ^[Yy]$ ]]; then
    echo ""
    echo "开始构建多平台镜像..."
    echo "这可能需要较长时间，请耐心等待..."
    echo ""
    
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t ${IMAGE_NAME}:${VERSION} \
        -t ${IMAGE_NAME}:latest \
        -f ${DOCKERFILE} \
        --load \
        .
else
    echo ""
    echo "开始构建当前平台镜像..."
    echo ""
    
    docker build \
        -t ${IMAGE_NAME}:${VERSION} \
        -t ${IMAGE_NAME}:latest \
        -f ${DOCKERFILE} \
        .
fi

echo ""
echo "======================================"
echo "构建完成!"
echo "======================================"
echo ""
echo "镜像标签:"
echo "  - ${IMAGE_NAME}:${VERSION}"
echo "  - ${IMAGE_NAME}:latest"
echo ""
echo "查看镜像："
echo "  docker images | grep ${IMAGE_NAME}"
echo ""
echo "导出镜像："
echo "  docker save ${IMAGE_NAME}:latest -o one-api-${VERSION}.tar"
echo ""
echo "运行容器："
echo "  cd docker && docker-compose up -d"
echo ""
