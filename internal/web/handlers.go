package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yufei/chaxin/internal/githubx"
	"github.com/yufei/chaxin/internal/notifier"
	"github.com/yufei/chaxin/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var in store.Settings
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	if err := s.store.SaveSettings(in); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 保存后校验 GitHub token（可选，仅当填写了 token）
	verify := map[string]any{"token_valid": true}
	if in.GitHubToken != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		client, err := githubx.NewClient(in.GitHubToken, in.GitHubAPIBaseURL)
		if err == nil {
			username, err := client.VerifyToken(ctx)
			if err != nil {
				verify["token_valid"] = false
				verify["token_error"] = err.Error()
			} else {
				verify["username"] = username
			}
		} else {
			verify["token_valid"] = false
			verify["token_error"] = err.Error()
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"settings": in,
		"verify":   verify,
	})
}

func (s *Server) handleSyncStars(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st.GitHubToken == "" {
		writeErr(w, http.StatusBadRequest, "请先在设置中配置 GitHub Token")
		return
	}
	client, err := githubx.NewClient(st.GitHubToken, st.GitHubAPIBaseURL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	stars, err := client.ListStarredRepos(ctx)
	if err != nil {
		if errors.Is(err, githubx.ErrUnauthorized) {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "同步失败: "+err.Error())
		return
	}

	added := 0
	for _, sr := range stars {
		isNew, err := s.store.UpsertRepo(store.Repo{
			FullName:    sr.FullName,
			Owner:       sr.Owner,
			Name:        sr.Name,
			Description: sr.Description,
			Language:    sr.Language,
			Stargazers:  sr.Stargazers,
			HTMLURL:     sr.HTMLURL,
		})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if isNew {
			added++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total": len(stars),
		"added": added,
	})
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.RepoFilter{
		Query:    q.Get("query"),
		Language: q.Get("language"),
	}
	switch q.Get("monitored") {
	case "1":
		t := true
		f.Monitored = &t
	case "0":
		t := false
		f.Monitored = &t
	}
	repos, err := s.store.ListRepos(f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if repos == nil {
		repos = []store.Repo{}
	}
	writeJSON(w, http.StatusOK, repos)
}

func (s *Server) handleAddRepo(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FullName string `json:"full_name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	in.FullName = strings.TrimSpace(in.FullName)
	parts := strings.Split(in.FullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeErr(w, http.StatusBadRequest, "仓库格式应为 owner/repo")
		return
	}
	if exists, err := s.store.RepoExists(in.FullName); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	} else if exists {
		writeErr(w, http.StatusConflict, "该仓库已存在")
		return
	}

	st, _ := s.store.GetSettings()
	client, err := githubx.NewClient(st.GitHubToken, st.GitHubAPIBaseURL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	info, err := client.RepoInfo(ctx, parts[0], parts[1])
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无法获取仓库信息: "+err.Error())
		return
	}
	if err := s.store.AddRepo(store.Repo{
		FullName:    info.FullName,
		Owner:       info.Owner,
		Name:        info.Name,
		Description: info.Description,
		Language:    info.Language,
		Stargazers:  info.Stargazers,
		HTMLURL:     info.HTMLURL,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"full_name": info.FullName})
}

func (s *Server) handlePatchRepo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的仓库 id")
		return
	}
	var in struct {
		Monitored *bool `json:"monitored"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if in.Monitored == nil {
		writeErr(w, http.StatusBadRequest, "缺少 monitored 字段")
		return
	}
	if _, err := s.store.GetRepoByID(id); err != nil {
		writeErr(w, http.StatusNotFound, "仓库不存在")
		return
	}
	if err := s.store.SetRepoMonitored(id, *in.Monitored); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"monitored": *in.Monitored})
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的仓库 id")
		return
	}
	if err := s.store.DeleteRepo(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := s.store.ListNotifications(limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []store.Notification{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	_ = decodeJSON(r, &in)
	if in.Title == "" {
		in.Title = "Chaxin 测试通知"
	}
	if in.Message == "" {
		in.Message = "如果你收到这条消息，说明通知配置已生效。"
	}

	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st.ShoutrrrURL == "" {
		writeErr(w, http.StatusBadRequest, "请先在设置中配置 Shoutrrr URL")
		return
	}
	n, err := notifier.New(st.ShoutrrrURL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := n.Send(in.Title, in.Message); err != nil {
		writeErr(w, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleRunMonitor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	if err := s.monitor.CheckAll(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "done"})
}
