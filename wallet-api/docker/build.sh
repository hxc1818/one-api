#!/bin/bash

# Docker 镜像构建脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 镜像信息
IMAGE_NAME="${IMAGE_NAME:-wallet-api}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-}"  # 例如: docker.io/username

echo "=========================================="
echo "  Wallet API Docker Image Builder"
echo "=========================================="
echo ""

# 显示构建信息
echo "Build Information:"
echo "  Image Name: $IMAGE_NAME"
echo "  Image Tag:  $IMAGE_TAG"
if [ -n "$REGISTRY" ]; then
    echo "  Registry:   $REGISTRY"
    FULL_IMAGE="$REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
else
    FULL_IMAGE="$IMAGE_NAME:$IMAGE_TAG"
fi
echo "  Full Image: $FULL_IMAGE"
echo ""

# 询问是否继续
read -p "Continue with build? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Build cancelled"
    exit 0
fi

# 构建镜像
echo ""
echo "Building Docker image..."
cd "$PROJECT_ROOT"

docker build \
    -f docker/Dockerfile \
    -t "$FULL_IMAGE" \
    --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    --build-arg VCS_REF="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
    .

echo ""
echo "✓ Build completed successfully!"
echo ""
echo "Image: $FULL_IMAGE"
echo "Size:  $(docker images $FULL_IMAGE --format '{{.Size}}')"
echo ""

# 询问是否推送到 Registry
if [ -n "$REGISTRY" ]; then
    read -p "Push to registry? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Pushing image to registry..."
        docker push "$FULL_IMAGE"
        echo "✓ Image pushed successfully!"
    fi
fi

# 显示使用说明
echo ""
echo "=========================================="
echo "  Usage"
echo "=========================================="
echo ""
echo "Run container:"
echo "  docker run -d \\"
echo "    -p 8081:8081 \\"
echo "    -e SQL_DSN='your_dsn' \\"
echo "    -e REDIS_CONN_STRING='redis://redis:6379/0' \\"
echo "    -e JWT_SECRET='your_secret' \\"
echo "    --name wallet-api \\"
echo "    $FULL_IMAGE"
echo ""
echo "Using docker-compose:"
echo "  cd docker"
echo "  docker-compose up -d"
echo ""
echo "=========================================="
