# 运行时镜像：二进制由 CI 预编译并下载到 bin/ 后拼装。
# 前端是纯静态资源、已通过 go:embed 内嵌进二进制，镜像内不再需要 Node 或 Go。
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --chmod=755 bin/chaxin /usr/local/bin/chaxin

ENV DATA_DIR=/data \
    LISTEN_ADDR=:8080

VOLUME ["/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/chaxin"]
