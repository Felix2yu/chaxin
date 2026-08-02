package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
	"github.com/yufei/chaxin/internal/web"
)

func main() {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	dataDir := envOr("DATA_DIR", "/data")
	addr := envOr("LISTEN_ADDR", ":8080")

	st, err := store.Open(dataDir)
	if err != nil {
		logger.Error("初始化存储失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := applyEnvDefaults(st); err != nil {
		logger.Error("应用环境变量默认配置失败", "err", err)
		os.Exit(1)
	}

	mon := monitor.New(st, logger)
	srv := web.NewServer(st, mon, logger)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go mon.Run(ctx)

	logger.Info("察新已启动", "addr", addr, "data_dir", dataDir, "log_level", level)

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务异常退出", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("收到退出信号，正在优雅关闭...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("HTTP 服务关闭异常", "err", err)
	}
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// applyEnvDefaults 将环境变量作为首次启动的默认配置写入数据库（不覆盖已有值）。
func applyEnvDefaults(st *store.Store) error {
	defaults := map[string]string{
		store.KeyGitHubToken:      os.Getenv("GITHUB_TOKEN"),
		store.KeyShoutrrrURL:      os.Getenv("SHOUTRRR_URL"),
		store.KeyPollInterval:     os.Getenv("POLL_INTERVAL"),
		store.KeyGitHubAPIBaseURL: os.Getenv("GITHUB_API_BASE_URL"),
	}
	if v := os.Getenv("NOTIFY_ON_FIRST_RUN"); v == "1" || v == "true" {
		defaults[store.KeyNotifyFirstRun] = "1"
	}
	for k, v := range defaults {
		if v == "" {
			continue
		}
		existing, err := st.GetSetting(k)
		if err != nil {
			return err
		}
		if existing == "" {
			if err := st.SetSetting(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
