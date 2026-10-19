# 运行时镜像：二进制由 CI 预编译并下载到 bin/ 后拼装。
# 前端是纯静态资源、已通过 go:embed 内嵌进二进制，镜像内不再需要 Node 或 Go。
#
# 基镜 ghcr.io/felix2yu/base:alpine-3.24 已提供 ca-certificates、tzdata、su-exec
# 与 PUID/PGID 降权入口（ENTRYPOINT /usr/local/bin/docker-entrypoint.sh）。
# 契约见 Felix2yu/.github/docs/docker.md。
FROM ghcr.io/felix2yu/base:alpine-3.24

# 容器内布局（全 fleet 统一）：二进制放 /app/chaxin，数据放 /app/data 挂载卷。
# /app 不在 PATH 上，所以外部备份工具的 BACKUP_SQLITE_CMD 必须写绝对路径
# `/app/chaxin backup`，裸名在 docker exec 里会 command not found。
# /app/data 由基镜入口按 PUID/PGID chown。
WORKDIR /app

COPY --chmod=755 bin/chaxin /app/chaxin

RUN mkdir -p /app/data

ENV DATA_DIR=/app/data \
    LISTEN_ADDR=:8080 \
    APP_DATA_DIR=/app/data

VOLUME ["/app/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

# ENTRYPOINT 由基镜提供（PUID/PGID 降权），这里只给服务命令
CMD ["/app/chaxin"]
