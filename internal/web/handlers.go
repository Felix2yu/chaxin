package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yufei/chaxin/internal/githubx"
	"github.com/yufei/chaxin/internal/notifier"
	"github.com/yufei/chaxin/internal/store"
	"github.com/yufei/chaxin/internal/translate"
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

	s.syncMu.Lock()
	if s.sync != nil && s.sync.Running {
		s.syncMu.Unlock()
		writeErr(w, http.StatusConflict, "同步正在进行中")
		return
	}
	s.sync = &SyncState{Running: true}
	s.syncMu.Unlock()

	go s.runSync(st)

	writeJSON(w, http.StatusOK, map[string]bool{"started": true})
}

func (s *Server) handleSyncStarsStatus(w http.ResponseWriter, r *http.Request) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.sync == nil {
		writeJSON(w, http.StatusOK, SyncState{})
		return
	}
	writeJSON(w, http.StatusOK, *s.sync)
}

// runSync 在后台执行 star 仓库同步，实时更新进度状态。
func (s *Server) runSync(st store.Settings) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := githubx.NewClient(st.GitHubToken, st.GitHubAPIBaseURL)
	if err != nil {
		s.finishSync(&SyncState{Error: err.Error()})
		return
	}

	keep := map[string]struct{}{}
	_, err = client.ListStarredReposPaged(ctx, func(page []githubx.StarredRepo, pageNum, totalPages int) error {
		added, updated, skipped := 0, 0, 0
		for _, sr := range page {
			keep[sr.FullName] = struct{}{}
			res, err := s.store.UpsertRepo(store.Repo{
				FullName:    sr.FullName,
				Owner:       sr.Owner,
				Name:        sr.Name,
				Description: sr.Description,
				Language:    sr.Language,
				Stargazers:  sr.Stargazers,
				HTMLURL:     sr.HTMLURL,
			})
			if err != nil {
				return err
			}
			switch res {
			case store.UpsertInserted:
				added++
			case store.UpsertUpdated:
				updated++
			case store.UpsertSkipped:
				skipped++
			}
		}
		s.syncMu.Lock()
		if s.sync != nil {
			s.sync.Page = pageNum
			s.sync.Total = totalPages
			s.sync.Progress = float64(pageNum) / float64(totalPages)
			if s.sync.Progress > 1 {
				s.sync.Progress = 1
			}
			s.sync.Repos += len(page)
			s.sync.Added += added
			s.sync.Updated += updated
			s.sync.Skipped += skipped
		}
		s.syncMu.Unlock()
		return nil
	})

	if err != nil {
		s.finishSync(&SyncState{Error: err.Error()})
		return
	}

	// 清理已取消 Star 的仓库（仅限 star 来源，手动添加的保留）
	removed, err := s.store.DeleteStarReposNotIn(keep)
	if err != nil {
		s.finishSync(&SyncState{Error: err.Error()})
		return
	}
	s.finishSync(&SyncState{Removed: int(removed)})
}

func (s *Server) finishSync(state *SyncState) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.sync == nil {
		return
	}
	if state != nil {
		s.sync.Error = state.Error
		s.sync.Removed += state.Removed
	}
	if s.sync.Progress == 0 {
		s.sync.Progress = 1
	}
	s.sync.Running = false
}

func (s *Server) handleBatchMonitor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs       []int64 `json:"ids"`
		Monitored *bool   `json:"monitored"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if in.Monitored == nil {
		writeErr(w, http.StatusBadRequest, "缺少 monitored 字段")
		return
	}
	if len(in.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, "缺少 ids 字段")
		return
	}
	updated, err := s.store.SetReposMonitored(in.IDs, *in.Monitored)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": updated, "monitored": *in.Monitored})
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
		Monitored     *bool   `json:"monitored"`
		IgnorePattern *string `json:"ignore_pattern"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if in.Monitored == nil && in.IgnorePattern == nil {
		writeErr(w, http.StatusBadRequest, "缺少要更新的字段")
		return
	}
	if _, err := s.store.GetRepoByID(id); err != nil {
		writeErr(w, http.StatusNotFound, "仓库不存在")
		return
	}
	if in.Monitored != nil {
		if err := s.store.SetRepoMonitored(id, *in.Monitored); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if in.IgnorePattern != nil {
		if err := s.store.SetRepoIgnorePattern(id, *in.IgnorePattern); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"monitored": in.Monitored, "ignore_pattern": in.IgnorePattern})
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的仓库 id")
		return
	}
	repo, err := s.store.GetRepoByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "仓库不存在")
		return
	}

	// 同时在 GitHub 上取消星标
	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	parts := strings.SplitN(repo.FullName, "/", 2)
	if len(parts) != 2 {
		writeErr(w, http.StatusInternalServerError, "仓库名格式异常")
		return
	}
	if st.GitHubToken != "" {
		client, err := githubx.NewClient(st.GitHubToken, st.GitHubAPIBaseURL)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := client.Unstar(ctx, parts[0], parts[1]); err != nil {
			writeErr(w, http.StatusBadGateway, "取消星标失败: "+err.Error())
			return
		}
	}
	if err := s.store.DeleteRepo(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	f := store.NotificationFilter{Limit: limit, Query: q.Get("query"), Status: q.Get("status")}
	items, err := s.store.ListNotifications(f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []store.Notification{}
	}
	writeJSON(w, http.StatusOK, items)
}

// handleRetryNotification 重新发送一条失败的通知记录。
func (s *Server) handleRetryNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的通知 id")
		return
	}
	items, err := s.store.ListNotifications(store.NotificationFilter{Limit: 1})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var target *store.Notification
	for i := range items {
		if items[i].ID == id {
			target = &items[i]
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusNotFound, "通知记录不存在")
		return
	}
	if target.Status == "sent" {
		writeErr(w, http.StatusBadRequest, "该通知已发送成功，无需重发")
		return
	}

	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st.ShoutrrrURL == "" {
		writeErr(w, http.StatusBadRequest, "请先配置 Shoutrrr URL")
		return
	}
	n, err := notifier.New(st.ShoutrrrURL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	title := fmt.Sprintf("%s 发布新版本 %s", target.FullName, target.Tag)
	msg := fmt.Sprintf("仓库: %s\n版本: %s\n链接: %s", target.FullName, target.Tag, target.ReleaseURL)
	// 重发优先使用已存译文，无译文则用原文
	body := strings.TrimSpace(target.ReleaseBodyTranslated)
	if body == "" {
		body = strings.TrimSpace(target.ReleaseBody)
	}
	if body != "" {
		msg += fmt.Sprintf("\n\n更新日志:\n%s", body)
	}
	if err := n.Send(title, msg); err != nil {
		writeErr(w, http.StatusInternalServerError, "重发失败: "+err.Error())
		return
	}
	if err := s.store.MarkNotificationSent(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	_ = decodeJSON(r, &in)
	if in.Title == "" {
		in.Title = "察新 测试通知"
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

// handleTranslate 按需翻译一段文本（前端语言切换时调用）。
func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text       string `json:"text"`
		TargetLang string `json:"target_lang"`
		Engine     string `json:"engine"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	if strings.TrimSpace(in.Text) == "" {
		writeErr(w, http.StatusBadRequest, "缺少待翻译文本")
		return
	}

	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	engine := in.Engine
	if engine == "" {
		engine = st.TranslateEngine
	}
	if engine == "" || engine == "off" {
		writeErr(w, http.StatusBadRequest, "请先在设置中配置翻译引擎")
		return
	}
	target := in.TargetLang
	if target == "" {
		target = st.TranslateTargetLang
	}

	cfg := translate.Config{
		Engine: engine,
		URL:    st.TranslateURL,
		APIKey: st.TranslateAPIKey,
		Model:  st.TranslateModel,
		Target: target,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	res, err := translate.Prepare(ctx, cfg, translate.Clean(in.Text))
	if err != nil {
		writeErr(w, http.StatusBadGateway, "翻译失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"translated": res.Translated,
		"extracted":  res.Extracted,
		"text":       res.Text,
	})
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

// Backup 导出的完整配置结构。
type Backup struct {
	Version   int             `json:"version"`
	Settings  store.Settings  `json:"settings"`
	Repos     []store.Repo    `json:"repos"`
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	repos, err := s.store.ListRepos(store.RepoFilter{})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if repos == nil {
		repos = []store.Repo{}
	}
	writeJSON(w, http.StatusOK, Backup{Version: 1, Settings: st, Repos: repos})
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	var in Backup
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的备份数据: "+err.Error())
		return
	}
	if in.Version < 1 {
		writeErr(w, http.StatusBadRequest, "备份数据格式不正确")
		return
	}
	if err := s.store.Restore(in.Settings, in.Repos); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}
