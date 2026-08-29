.PHONY: dev build run docker up down clean

# 前端为纯静态资源（HTML/CSS/JS），由 build.sh 复制到 internal/web/dist 供 Go 内嵌。
# 无需 Node / 打包器，构建极轻量。

# 开发：构建前端 + 启动后端（:8080）
dev: build
	DATA_DIR=./data LISTEN_ADDR=:8080 ./bin/chaxin

# 构建：先构建前端静态资源，再编译后端到 bin/
build:
	sh web/build.sh
	go build -o bin/chaxin ./cmd/server

# 本地运行（先构建）
run: build
	DATA_DIR=./data LISTEN_ADDR=:8080 ./bin/chaxin

# 构建 Docker 镜像
docker:
	docker build -t chaxin:latest .

# 通过 docker compose 构建并启动
up:
	docker compose up -d --build

# 停止
down:
	docker compose down

clean:
	rm -rf bin data
	rm -rf internal/web/dist/*
