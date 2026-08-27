# syntax=docker/dockerfile:1
# ---- build stage ----
FROM golang:1.23-alpine AS build

WORKDIR /src

# 先复制模块文件，利用 Docker layer cache；本项目零第三方依赖。
COPY go.mod ./
COPY . .

# CGO 关闭，静态二进制可重复构建。
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/grid-fault-locate-service .

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /data \
    && chown -R app:app /data

WORKDIR /app
COPY --from=build /out/grid-fault-locate-service /app/grid-fault-locate-service

ENV PORT=8080 \
    DATA_FILE=/data/grid-fault-locate-data.json \
    PERSIST=true \
    LOG_LEVEL=info

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1

ENTRYPOINT ["/app/grid-fault-locate-service"]
