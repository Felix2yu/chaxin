.PHONY: dev build run docker up down clean

# 开发：前端热更新 + 后端热重载（后端 :8080，前端 :5173 代理到后端）
dev:
	cd web && npm run dev &
	@echo "等待前端启动后，请手动运行: go run ./cmd/server"

# 构建：先构建前端，再编译后端到 bin/
build:
	cd web && npm run build
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
	rm -rf web/node_modules web/dist internal/web/dist/*
