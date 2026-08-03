package web

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yufei/chaxin/internal/store"
)

// rssItem 单条发布条目。
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// rssChannel RSS 2.0 频道。
type rssChannel struct {
	XMLName     xml.Name  `xml:"rss"`
	Version     string    `xml:"version,attr"`
	Channel     channel   `xml:"channel"`
}

type channel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	LastBuildDate string  `xml:"lastBuildDate"`
	Items       []rssItem `xml:"item"`
}

// handleFeed 输出聚合所有被监控仓库最新版本的 RSS 订阅源。
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	repos, err := s.store.ListRepos(store.RepoFilter{})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 仅保留被监控且已缓存最新版本的仓库
	items := make([]rssItem, 0, len(repos))
	for _, repo := range repos {
		if !repo.Monitored || repo.LatestTag == "" {
			continue
		}
		title := fmt.Sprintf("%s 发布 %s", repo.FullName, repo.LatestTag)
		link := repo.LatestReleaseURL
		if link == "" {
			link = repo.HTMLURL
		}
		desc := repo.LatestReleaseBody
		if desc == "" {
			desc = repo.Description
		}
		item := rssItem{
			Title:       title,
			Link:        link,
			GUID:        fmt.Sprintf("%s@%s", repo.FullName, repo.LatestTag),
			Description: feedDescription(desc),
		}
		if !repo.LatestReleaseAt.IsZero() {
			item.PubDate = repo.LatestReleaseAt.UTC().Format(time.RFC1123Z)
		}
		items = append(items, item)
	}

	// 按发布时间倒序
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].PubDate > items[j].PubDate
	})

	// 自引用地址（供阅读器补全订阅）
	base := feedBaseURL(r)

	out := rssChannel{
		Version: "2.0",
		Channel: channel{
			Title:         "察新 · GitHub Release 监控",
			Link:          base,
			Description:   "聚合所有被监控仓库的最新版本发布",
			LastBuildDate: time.Now().UTC().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(out)
}

// feedBaseURL 从请求推导站点根地址。
func feedBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	return scheme + "://" + host
}

// feedDescription 截断更新日志作为 RSS 描述，避免超长条目。
func feedDescription(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "暂无更新日志"
	}
	runes := []rune(body)
	const maxDesc = 1000
	if len(runes) <= maxDesc {
		return body
	}
	return string(runes[:maxDesc]) + "\n…（更新日志过长已截断，完整内容见链接）"
}
