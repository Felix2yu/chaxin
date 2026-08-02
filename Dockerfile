# ---------- 阶段 1：前端构建 ----------
FROM node:26-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- 阶段 2：后端构建 ----------
FROM golang:1.26-alpine AS server-builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /chaxin ./cmd/server

# ---------- 阶段 3：运行 ----------
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=server-builder /chaxin /usr/local/bin/chaxin
ENV DATA_DIR=/data \
    LISTEN_ADDR=:8080
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/chaxin"]
