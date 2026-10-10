package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
)

type Server struct {
	store   *store.Store
	monitor *monitor.Monitor
	logger  *slog.Logger
	mux     *http.ServeMux

	syncMu sync.Mutex
	sync   *SyncState
}

// SyncState 记录 star 同步任务进度。
type SyncState struct {
	Running  bool    `json:"running"`
	Page     int     `json:"page"`
	Total    int     `json:"total"`
	Progress float64 `json:"progress"` // 0.0 ~ 1.0
	Repos    int     `json:"repos"`    // 已处理的仓库数
	Added    int     `json:"added"`    // 新增
	Updated  int     `json:"updated"`  // 信息有更新
	Skipped  int     `json:"skipped"`  // 无变化
	Removed  int     `json:"removed"`  // 已取消 Star 被移除
	Error    string  `json:"error"`
}

func NewServer(st *store.Store, mon *monitor.Monitor, logger *slog.Logger) *Server {
	s := &Server{
		store:   st,
		monitor: mon,
		logger:  logger,
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /api/health", s.handleHealth)
	m.HandleFunc("GET /api/settings", s.handleGetSettings)
	m.HandleFunc("PUT /api/settings", s.handlePutSettings)
	m.HandleFunc("POST /api/repos/sync-stars", s.handleSyncStars)
	m.HandleFunc("GET /api/repos/sync-stars/status", s.handleSyncStarsStatus)
	m.HandleFunc("GET /api/repos", s.handleListRepos)
	m.HandleFunc("POST /api/repos", s.handleAddRepo)
	m.HandleFunc("PATCH /api/repos/{id}", s.handlePatchRepo)
	m.HandleFunc("POST /api/repos/batch-monitor", s.handleBatchMonitor)
	m.HandleFunc("DELETE /api/repos/{id}", s.handleDeleteRepo)
	m.HandleFunc("GET /api/notifications", s.handleListNotifications)
	m.HandleFunc("POST /api/notifications/{id}/retry", s.handleRetryNotification)
	m.HandleFunc("POST /api/translate", s.handleTranslate)
	m.HandleFunc("POST /api/test-notification", s.handleTestNotification)
	m.HandleFunc("GET /feed", s.handleFeed)
	m.HandleFunc("GET /rss", s.handleFeed)
	m.HandleFunc("POST /api/monitor/run", s.handleRunMonitor)
	m.HandleFunc("GET /api/backup", s.handleBackup)
	m.HandleFunc("POST /api/restore", s.handleRestore)
	// PWA 清单与 Service Worker 显式路由：Go 的 mime 表不认识 .webmanifest，
	// 交给 http.FileServer 会落到 text/plain，浏览器可能拒绝解析。
	m.HandleFunc("GET /manifest.webmanifest", s.handleManifest)
	m.HandleFunc("GET /sw.js", s.handleServiceWorker)
	m.HandleFunc("/", s.handleStatic)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(sw, r)
	s.logger.Info("http", "method", r.Method, "path", r.URL.Path,
		"status", sw.status, "duration", time.Since(start).Round(time.Millisecond))
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "接口不存在"})
		return
	}
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		http.Error(w, "前端资源未构建", http.StatusServiceUnavailable)
		return
	}
	// embed.FS 的文件没有修改时间，http.FileServer 不会发出 Last-Modified/ETag，
	// 浏览器只能启发式缓存旧 JS，导致部署新版后页面仍用旧脚本、与后端 API 不匹配。
	// 显式 no-cache 强制每次回源取最新资源（前端总体积仅数十 KB，代价可忽略）。
	w.Header().Set("Cache-Control", "no-cache")
	fsrv := http.FileServer(http.FS(sub))
	clean := strings.TrimPrefix(r.URL.Path, "/")
	if clean != "" {
		if _, err := sub.Open(clean); err != nil {
			// SPA fallback：将路由交给前端入口
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fsrv.ServeHTTP(w, r2)
			return
		}
	}
	fsrv.ServeHTTP(w, r)
}

// handleManifest 下发 Web App Manifest。Go 的 mime 表没有 .webmanifest 的映射，
// 若不显式指定，http.FileServer 会按内容嗅探出 text/plain。
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	s.serveDistFile(w, r, "manifest.webmanifest", "application/manifest+json", "public, max-age=3600")
}

// handleServiceWorker 以 no-cache 下发 SW：部署后浏览器下次导航即可拿到新版本，
// 新 SW 通过版本化缓存名淘汰旧缓存（见 web/sw.js 的行为契约）。
func (s *Server) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	s.serveDistFile(w, r, "sw.js", "application/javascript; charset=utf-8", "no-cache")
}

func (s *Server) serveDistFile(w http.ResponseWriter, r *http.Request, name, contentType, cacheControl string) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		http.Error(w, "前端资源未构建", http.StatusServiceUnavailable)
		return
	}
	data, err := fs.ReadFile(sub, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	_, _ = w.Write(data)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
