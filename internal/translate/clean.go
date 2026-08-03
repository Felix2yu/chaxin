package translate

import (
	"regexp"
	"strings"
)

// 预编译正则：HTML 块 / 标签 / markdown 链接 / 裸 URL / commit SHA。
var (
	// 成对 HTML 标签块（含嵌套，逐层剥除）。
	htmlBlockRe = regexp.MustCompile(`(?s)<(\w+)[^>]*>.*?</\w+>`)
	// 自闭合或孤立的 HTML 标签。
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	// markdown 链接 [text](url) 整体移除。
	mdLinkRe = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)
	// 裸 URL。
	rawURLRe = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"')]+`)
	// commit SHA（7-40 位十六进制，覆盖短 SHA 与完整 SHA）。
	shaRe = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)
	// 连续空行（3 个及以上换行）压缩。
	blankLineRe = regexp.MustCompile(`\n{3,}`)
)

// Clean 清洗更新日志正文：移除 HTML 块与标签、markdown 链接、裸 URL、
// commit SHA，并压缩多余空行。清洗在翻译与存储之前执行，避免噪音进入
// 通知消息、记录页与 RSS。
func Clean(body string) string {
	s := body

	// 先移除 markdown 链接（避免链接文字干扰后续匹配）
	s = mdLinkRe.ReplaceAllString(s, "")
	// 反复移除成对 HTML 块（含嵌套，从内向外逐层剥除）
	for {
		next := htmlBlockRe.ReplaceAllString(s, "")
		if next == s {
			break
		}
		s = next
	}
	// 移除残留的自闭合/孤立标签
	s = htmlTagRe.ReplaceAllString(s, "")
	// 移除裸 URL
	s = rawURLRe.ReplaceAllString(s, "")
	// 移除 commit SHA
	s = shaRe.ReplaceAllString(s, "")

	// 清理空白：逐行去首尾空格，压缩连续空行
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimSpace(ln)
	}
	s = strings.Join(lines, "\n")
	s = blankLineRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
