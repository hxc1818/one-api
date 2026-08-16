#!/bin/bash

# One API 快速启动脚本
# 使用方法: ./start.sh [sqlite|compose|rebuild]

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    print_info "Docker 版本: $(docker --version)"
}

# 启动单容器模式（SQLite）
start_sqlite() {
    print_info "启动 One API（SQLite 模式）..."
    
    # 创建数据目录
    mkdir -p ./data
    
    # 生成随机 SESSION_SECRET
    SESSION_SECRET=$(openssl rand -hex 16 2>/dev/null || echo "please_change_this_secret_$(date +%s)")
    
    # 检查容器是否已存在
    if docker ps -a | grep -q "one-api"; then
        print_warn "容器 one-api 已存在，正在删除..."
        docker rm -f one-api
    fi
    
    # 启动容器
    docker run -d \
        --name one-api \
        --restart always \
        -p 3000:3000 \
        -e TZ=Asia/Shanghai \
        -e SESSION_SECRET="${SESSION_SECRET}" \
        -v "$(pwd)/data:/data" \
        one-api:latest
    
    print_info "容器已启动！"
    print_info "访问地址: http://localhost:3000"
    print_info "默认用户名: root"
    print_info "默认密码: 123456"
    print_warn "请登录后立即修改密码！"
    print_info "SESSION_SECRET: ${SESSION_SECRET}"
    
    # 等待服务启动
    sleep 3
    docker logs one-api --tail 20
}

# 使用 Docker Compose 启动
start_compose() {
    print_info "使用 Docker Compose 启动完整服务..."
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose 未安装"
        exit 1
    fi
    
    # 创建数据目录
    mkdir -p ./data/oneapi ./data/mysql ./logs
    
    # 启动服务
    if docker compose version &> /dev/null; then
        docker compose up -d
    else
        docker-compose up -d
    fi
    
    print_info "服务已启动！"
    print_info "访问地址: http://localhost:3000"
    print_info "默认用户名: root"
    print_info "默认密码: 123456"
    print_warn "请登录后立即修改密码！"
    
    print_info "查看日志: docker-compose logs -f one-api"
}

# 重新构建镜像
rebuild_image() {
    print_info "重新构建 Docker 镜像..."
    
    # 检查 Dockerfile
    if [ ! -f "Dockerfile" ]; then
        print_error "未找到 Dockerfile"
        exit 1
    fi
    
    # 设置版本号
    if [ -f ".git/HEAD" ]; then
        VERSION=$(git describe --tags --always 2>/dev/null || git rev-parse --short HEAD)
        echo "$VERSION" > VERSION
        print_info "版本号: $VERSION"
    else
        echo "v0.0.1" > VERSION
    fi
    
    # 构建镜像
    docker build -t one-api:latest -f Dockerfile .
    
    print_info "镜像构建完成！"
    docker images | grep one-api
}

# 显示状态
show_status() {
    print_info "=== 容器状态 ==="
    docker ps -a | grep -E "CONTAINER|one-api|mysql|redis" || echo "没有运行的容器"
    
    echo ""
    print_info "=== 镜像信息 ==="
    docker images | grep -E "REPOSITORY|one-api" || echo "未找到 one-api 镜像"
}

# 停止服务
stop_services() {
    print_info "停止服务..."
    
    # 停止单容器
    if docker ps | grep -q "one-api"; then
        docker stop one-api
        print_info "已停止 one-api 容器"
    fi
    
    # 停止 compose 服务
    if [ -f "docker-compose.yml" ]; then
        if docker compose version &> /dev/null; then
            docker compose stop
        else
            docker-compose stop
        fi
        print_info "已停止 Docker Compose 服务"
    fi
}

# 显示帮助
show_help() {
    cat << EOF
One API Docker 启动脚本

用法: $0 [命令]

命令:
  sqlite      使用 SQLite 启动单容器（默认）
  compose     使用 Docker Compose 启动完整服务（MySQL + Redis）
  rebuild     重新构建 Docker 镜像
  status      显示当前状态
  stop        停止所有服务
  logs        查看日志
  help        显示帮助信息

示例:
  $0              # 启动 SQLite 模式
  $0 compose      # 启动完整服务
  $0 rebuild      # 重新构建镜像
  $0 status       # 查看状态

EOF
}

# 查看日志
show_logs() {
    if docker ps | grep -q "one-api"; then
        print_info "显示 one-api 日志..."
        docker logs -f --tail 50 one-api
    else
        print_error "容器未运行"
        exit 1
    fi
}

# 主逻辑
main() {
    check_docker
    
    case "${1:-sqlite}" in
        sqlite)
            start_sqlite
            ;;
        compose)
            start_compose
            ;;
        rebuild)
            rebuild_image
            ;;
        status)
            show_status
            ;;
        stop)
            stop_services
            ;;
        logs)
            show_logs
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
