package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
)

type Server struct {
	store   *store.Store
	monitor *monitor.Monitor
	logger  *slog.Logger
	mux     *http.ServeMux
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
	m.HandleFunc("GET /api/repos", s.handleListRepos)
	m.HandleFunc("POST /api/repos", s.handleAddRepo)
	m.HandleFunc("PATCH /api/repos/{id}", s.handlePatchRepo)
	m.HandleFunc("DELETE /api/repos/{id}", s.handleDeleteRepo)
	m.HandleFunc("GET /api/notifications", s.handleListNotifications)
	m.HandleFunc("POST /api/test-notification", s.handleTestNotification)
	m.HandleFunc("POST /api/monitor/run", s.handleRunMonitor)
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

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
