# 项目长期记忆

## 前端技术选型约定
- 用户明确要求：前端使用**纯 JavaScript（不使用 TypeScript）**，保持代码轻量、易维护。
- 当前实现：原生 JS + ES Modules，无框架、无打包器、无 npm 依赖。构建由 `web/build.sh` 复制到 `internal/web/dist`，Go `go:embed` 内嵌。
- 若需新增前端功能，沿用此约束：不要引入 Vue/React/TS/Vite 等重工具链。
- 后端（`cmd/`、`internal/`）保持 Go 不变；前端 API 契约见 `internal/web/handlers.go`。
