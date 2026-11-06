# syntax=docker/dockerfile:1

# -------------------------------------------------------------
# Stage 1: Build binary
# -------------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 安装必要的构建工具与依赖
RUN apk add --no-cache ca-certificates tzdata

# 依赖缓存优化
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译纯 Go 二进制（关闭 CGO，剥离调试符号减小体积）
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/dispatcher \
    ./cmd/dispatcher

# -------------------------------------------------------------
# Stage 2: Runtime image
# -------------------------------------------------------------
FROM alpine:3.20

WORKDIR /app

# 拷贝系统证书与时区数据库
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/dispatcher /app/dispatcher

# 创建数据卷挂载点并建立非 root 用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir -p /app/data \
    && chown -R appuser:appgroup /app

USER appuser

ENV ADDR=":8080" \
    DB_PATH="/app/data/dispatcher.db" \
    TZ="Asia/Shanghai"

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

VOLUME ["/app/data"]

ENTRYPOINT ["/app/dispatcher"]
