# Chaxin · GitHub Release 监控

监控你 Star 的 GitHub 仓库（或手动添加的任意仓库）的 Release 发布，发现新版本时通过 [shoutrrr](https://github.com/containrrr/shoutrrr) 发送通知。自带现代化 Web 管理界面，前后端打包为单个 Docker 容器。

## 功能特性

- **GitHub 认证**：使用 Personal Access Token（PAT）接入 GitHub API
- **同步 Star 仓库**：一键拉取认证用户 Star 的全部仓库并入本地库
- **手动添加仓库**：支持监控任意 `owner/repo`，不限于 Star 仓库
- **版本发布监控**：自动轮询仓库最新正式版（忽略 draft / prerelease），检测到新版本立即通知
- **更新日志**：通知消息与记录中同时携带 Release 更新日志（超长自动截断），前端可展开查看全文
- **通知**：通过 shoutrrr 支持 Telegram / Discord / Slack / 邮件等 40+ 服务
- **首轮基线**：默认首次监控只建立基线不通知，避免大量历史通知刷屏（可在设置中开启）
- **现代 Web UI**：概览仪表盘、仓库管理、通知记录、设置页
- **单容器部署**：Vue 前端构建产物由 Go 二进制内嵌（`go:embed`），一个容器即可运行

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.26 · go-github v89 · shoutrrr v0.8 · SQLite（纯 Go 无 CGO） |
| 前端 | Vue 3 · Vite · TypeScript · Tailwind CSS 4 · Pinia · Vue Router |
| 部署 | 多阶段 Docker 构建（node:26-alpine + golang:1.26-alpine + alpine） |

## 快速开始（Docker Compose）

### 从 GHCR 拉取镜像

GitHub Actions 会自动构建多架构（amd64 / arm64）镜像并推送到 GHCR：

```bash
# 拉取 main 分支最新构建
docker pull ghcr.io/felix2yu/chaxin:latest
```

若使用 `docker-compose.yml` 本地构建运行：

```bash
docker compose up -d --build
```

访问 <http://localhost:8080>，在「设置」页完成配置：

1. **GitHub Token**：在 GitHub [Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens) 创建，勾选 `repo` 与 `read:user` 权限
2. **Shoutrrr URL**：填通知目标地址，例如 Telegram：

   ```
   telegram://bot_token@telegram?channels=channel_id
   ```

   其他服务格式见 [shoutrrr 文档](https://containrrr.dev/shoutrrr/)
3. 设置检查间隔，保存后点击「发送测试通知」验证

然后在「仓库」页点击「同步我的 Star」，或手动添加仓库，打开对应仓库的监控开关即可。

## 环境变量

首次启动时，以下环境变量会作为数据库默认值写入（**仅在数据库尚无该值时生效**，之后以 Web 设置页为准）。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DATA_DIR` | `/data` | 数据目录（SQLite 存放于此，容器内挂载卷） |
| `LISTEN_ADDR` | `:8080` | HTTP 监听地址 |
| `GITHUB_TOKEN` | - | 首次启动默认的 GitHub PAT |
| `SHOUTRRR_URL` | - | 首次启动默认的通知 URL |
| `POLL_INTERVAL` | - | 轮询间隔，如 `5m`、`30m`、`1h` |
| `NOTIFY_ON_FIRST_RUN` | - | 是否在首次监控时通知历史最新版（`true`/`1`） |
| `GITHUB_API_BASE_URL` | - | GitHub Enterprise 的 API Base URL（默认官方 `https://api.github.com/`） |

## REST API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查 |
| GET/PUT | `/api/settings` | 读取 / 保存配置（保存时校验 token） |
| POST | `/api/repos/sync-stars` | 同步 Star 仓库 |
| GET | `/api/repos` | 仓库列表（支持 `query`/`language`/`monitored` 过滤） |
| POST | `/api/repos` | 手动添加仓库 |
| PATCH | `/api/repos/{id}` | 切换监控状态（`{"monitored": true}`） |
| DELETE | `/api/repos/{id}` | 删除仓库 |
| GET | `/api/notifications` | 通知记录（`limit`） |
| POST | `/api/test-notification` | 发送测试通知 |
| POST | `/api/monitor/run` | 立即执行一轮检查 |

## 本地开发

```bash
# 1. 启动后端（终端 A）
go run ./cmd/server

# 2. 启动前端开发服务器（终端 B，:5173 代理 /api 到 :8080）
cd web && npm install && npm run dev
```

访问 <http://localhost:5173>。前端构建产物输出到 `internal/web/dist`，由后端 `go:embed` 内嵌，因此本地改动前端后需 `cd web && npm run build` 才能在 `:8080` 生效。

## 项目结构

```
├── cmd/server/            # 入口
├── internal/
│   ├── store/             # SQLite 存储层
│   ├── githubx/           # GitHub 客户端封装
│   ├── monitor/           # 轮询调度器
│   ├── notifier/          # shoutrrr 通知封装
│   └── web/               # REST API + go:embed 静态服务
│       └── dist/          # 前端构建产物（自动生成）
├── web/                   # Vue 3 前端
├── Dockerfile             # 多阶段构建（node:26 / golang:1.26 / alpine）
└── docker-compose.yml
```

## 许可证

MIT
