#!/bin/bash

# Wallet API Docker 构建和部署脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  Wallet API Docker Deployment Script"
echo "=========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 函数：打印成功消息
success() {
    echo -e "${GREEN}✓${NC} $1"
}

# 函数：打印错误消息
error() {
    echo -e "${RED}✗${NC} $1"
}

# 函数：打印警告消息
warning() {
    echo -e "${YELLOW}!${NC} $1"
}

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

success "Docker and Docker Compose are installed"

# 检查 .env 文件
if [ ! -f "$SCRIPT_DIR/.env" ]; then
    warning ".env file not found. Creating from .env.example..."
    cp "$SCRIPT_DIR/.env.example" "$SCRIPT_DIR/.env"
    warning "Please edit $SCRIPT_DIR/.env and configure your settings"
    echo ""
    echo "Required settings:"
    echo "  - SQL_DSN: Database connection string"
    echo "  - REDIS_CONN_STRING: Redis connection string"
    echo "  - JWT_SECRET: Random secret key for JWT"
    echo ""
    read -p "Press Enter after configuring .env file..."
fi

success ".env file found"

# 加载环境变量
set -a
source "$SCRIPT_DIR/.env"
set +a

# 验证必需的环境变量
REQUIRED_VARS=("SQL_DSN" "REDIS_CONN_STRING" "JWT_SECRET")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    error "Missing required environment variables:"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    exit 1
fi

success "All required environment variables are set"

# 显示配置信息
echo ""
echo "Configuration:"
echo "  SQL_DSN: ${SQL_DSN:0:40}..."
echo "  REDIS_CONN_STRING: $REDIS_CONN_STRING"
echo "  WALLET_API_PORT: ${WALLET_API_PORT:-8081}"
echo "  JWT_SECRET: ********"
echo ""

# 询问操作
echo "Select operation:"
echo "  1) Build and start (full deployment)"
echo "  2) Build only"
echo "  3) Start services"
echo "  4) Stop services"
echo "  5) View logs"
echo "  6) Restart services"
echo "  7) Clean up (remove containers and volumes)"
echo ""
read -p "Enter your choice [1-7]: " choice

cd "$SCRIPT_DIR"

case $choice in
    1)
        echo ""
        echo "Building and starting Wallet API..."
        docker-compose down
        docker-compose build --no-cache
        docker-compose up -d
        success "Wallet API is now running"
        echo ""
        echo "Checking service health..."
        sleep 5
        docker-compose ps
        echo ""
        echo "View logs: docker-compose -f $SCRIPT_DIR/docker-compose.yml logs -f wallet-api"
        ;;
    2)
        echo ""
        echo "Building Docker image..."
        docker-compose build --no-cache
        success "Build completed"
        ;;
    3)
        echo ""
        echo "Starting services..."
        docker-compose up -d
        success "Services started"
        docker-compose ps
        ;;
    4)
        echo ""
        echo "Stopping services..."
        docker-compose down
        success "Services stopped"
        ;;
    5)
        echo ""
        echo "Viewing logs (Ctrl+C to exit)..."
        docker-compose logs -f wallet-api
        ;;
    6)
        echo ""
        echo "Restarting services..."
        docker-compose restart
        success "Services restarted"
        docker-compose ps
        ;;
    7)
        echo ""
        warning "This will remove all containers and volumes!"
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            docker-compose down -v
            success "Cleanup completed"
        else
            echo "Cancelled"
        fi
        ;;
    *)
        error "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "=========================================="
echo "  Wallet API Endpoints"
echo "=========================================="
echo ""
echo "  Health Check: http://localhost:${WALLET_API_PORT:-8081}/health"
echo "  API Base URL: http://localhost:${WALLET_API_PORT:-8081}/api/v1"
echo ""
echo "  Auth:         POST /api/v1/auth/verify"
echo "  Balance:      GET  /api/v1/balance"
echo "  Deduct:       POST /api/v1/deduct"
echo "  Refund:       POST /api/v1/refund"
echo "  Transactions: GET  /api/v1/transactions"
echo ""
echo "=========================================="
