#!/bin/bash

# Wallet API 部署脚本

set -e

echo "=== Wallet API Deployment Script ==="

# 检查环境变量
if [ -z "$SQL_DSN" ]; then
    echo "Error: SQL_DSN environment variable is not set"
    echo "Example: export SQL_DSN='root:password@tcp(localhost:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local'"
    exit 1
fi

if [ -z "$REDIS_CONN_STRING" ]; then
    echo "Error: REDIS_CONN_STRING environment variable is not set"
    echo "Example: export REDIS_CONN_STRING='redis://localhost:6379/0'"
    exit 1
fi

# 设置默认值
export JWT_SECRET=${JWT_SECRET:-"wallet-api-secret-$(openssl rand -hex 16)"}
export WALLET_API_PORT=${WALLET_API_PORT:-8081}
export REDIS_STREAM_KEY=${REDIS_STREAM_KEY:-"wallet:balance:updates"}
export REDIS_STREAM_MAXLEN=${REDIS_STREAM_MAXLEN:-10000}

echo "Environment variables:"
echo "  SQL_DSN: ${SQL_DSN:0:30}..."
echo "  REDIS_CONN_STRING: ${REDIS_CONN_STRING}"
echo "  WALLET_API_PORT: ${WALLET_API_PORT}"
echo "  JWT_SECRET: ********"
echo "  REDIS_STREAM_KEY: ${REDIS_STREAM_KEY}"
echo "  REDIS_STREAM_MAXLEN: ${REDIS_STREAM_MAXLEN}"

# 编译
echo ""
echo "=== Building Wallet API ==="
cd "$(dirname "$0")"
go mod download
go build -o wallet-api -ldflags="-s -w" .

echo ""
echo "=== Build completed successfully ==="
echo ""
echo "To start the service, run:"
echo "  ./wallet-api"
echo ""
echo "Or use systemd service (example):"
echo "  sudo cp wallet-api.service /etc/systemd/system/"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable wallet-api"
echo "  sudo systemctl start wallet-api"
