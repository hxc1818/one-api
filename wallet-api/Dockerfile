FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o wallet-api .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从 builder 复制可执行文件
COPY --from=builder /app/wallet-api .

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8081

CMD ["./wallet-api"]
