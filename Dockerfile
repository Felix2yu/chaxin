# ---------- 阶段 1：后端构建（前端为纯静态资源，直接拷贝） ----------
FROM golang:1.27-alpine AS server-builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 将纯静态前端复制到 internal/web/dist 供 go:embed 内嵌（无需 Node 打包器）
RUN sh web/build.sh
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /chaxin ./cmd/server

# ---------- 阶段 3：运行 ----------
FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=server-builder /chaxin /usr/local/bin/chaxin
ENV DATA_DIR=/data \
    LISTEN_ADDR=:8080
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/usr/local/bin/chaxin"]
