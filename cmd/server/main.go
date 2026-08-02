package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
	"github.com/yufei/chaxin/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	logger.Info("察新已启动", "addr", addr, "data_dir", dataDir)
	err = httpSrv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP 服务异常退出", "err", err)
		os.Exit(1)
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
