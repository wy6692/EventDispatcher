# -------------------------------------------------------------
# Stage 1: Build binary
# -------------------------------------------------------------
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.22-alpine AS builder

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
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:3.20

WORKDIR /app

# 拷贝系统证书与时区数据库
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/dispatcher /app/dispatcher

# 创建数据卷挂载点并建立非 root 用户。
# 挂载宿主机目录会盖掉镜像内的属主，入口脚本在启动时再修正权限。
RUN apk add --no-cache su-exec \
    && addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir -p /app/data \
    && chown -R appuser:appgroup /app \
    && printf '%s\n' \
        '#!/bin/sh' \
        'set -eu' \
        'mkdir -p /app/data' \
        'chown -R appuser:appgroup /app/data' \
        'exec su-exec appuser:appgroup /app/dispatcher' \
        > /docker-entrypoint.sh \
    && chmod 755 /docker-entrypoint.sh

ENV ADDR=":8080" \
    DB_PATH="/app/data/dispatcher.db" \
    TZ="Asia/Shanghai"

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

VOLUME ["/app/data"]

ENTRYPOINT ["/docker-entrypoint.sh"]
