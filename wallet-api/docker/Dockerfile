FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译（启用 CGO 以支持 SQLite）
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-s -w" -o wallet-api .

# 运行阶段 - 使用更小的基础镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

# 创建非 root 用户
RUN addgroup -g 1000 wallet && \
    adduser -D -u 1000 -G wallet wallet

WORKDIR /app

# 从 builder 复制可执行文件
COPY --from=builder /build/wallet-api .

# 设置权限
RUN chown -R wallet:wallet /app

# 切换到非 root 用户
USER wallet

# 设置时区
ENV TZ=Asia/Shanghai

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

EXPOSE 8081

CMD ["./wallet-api"]
