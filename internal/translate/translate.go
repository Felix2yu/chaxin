package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config 翻译引擎配置（来自 Settings 的 translate_* 字段）。
type Config struct {
	Engine  string // dlx / bing / openai
	URL     string // dlx 服务地址 或 openai base_url
	APIKey  string // openai 专用
	Model   string // openai 专用
	Target  string // 目标语言代码，如 ZH / zh-Hans / en
}

// ErrNotConfigured 表示未配置可用的翻译引擎。
var ErrNotConfigured = errors.New("未配置翻译引擎")

// httpClient 统一 HTTP 客户端（30s 超时）。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Translate 调用配置的引擎翻译单段文本，返回译文。
func Translate(ctx context.Context, cfg Config, text string) (string, error) {
	switch strings.ToLower(cfg.Engine) {
	case "dlx":
		return translateDLX(ctx, cfg, text)
	case "bing":
		return translateBing(ctx, cfg, text)
	case "google":
		return translateGoogle(ctx, cfg, text)
	case "openai":
		return translateOpenAI(ctx, cfg, text)
	default:
		return "", ErrNotConfigured
	}
}

// languageName 供 OpenAI prompt 使用的目标语言中文名。
func languageName(code string) string {
	switch normalizeLang(code) {
	case LangZhHant:
		return "繁体中文"
	case LangZhHans:
		return "简体中文"
	case LangEn:
		return "英文"
	case LangJa:
		return "日文"
	case LangKo:
		return "韩文"
	default:
		return code
	}
}

// dlxLang 将通用目标代码转为 DLX 识别的 DeepL 语言代码。
func dlxLang(code string) string {
	switch normalizeLang(code) {
	case LangZhHant:
		return "ZH-HANT"
	case LangZhHans:
		return "ZH"
	case LangEn:
		return "EN"
	case LangJa:
		return "JA"
	case LangKo:
		return "KO"
	default:
		return strings.ToUpper(code)
	}
}

// bingLang 将通用目标代码转为必应识别的语言代码。
func bingLang(code string) string {
	switch normalizeLang(code) {
	case LangZhHant:
		return "zh-Hant"
	case LangZhHans:
		return "zh-Hans"
	case LangEn:
		return "en"
	case LangJa:
		return "ja"
	case LangKo:
		return "ko"
	default:
		return code
	}
}

// googleLang 将通用目标代码转为 Google 翻译识别的语言代码。
func googleLang(code string) string {
	switch normalizeLang(code) {
	case LangZhHant:
		return "zh-TW"
	case LangZhHans:
		return "zh-CN"
	case LangEn:
		return "en"
	case LangJa:
		return "ja"
	case LangKo:
		return "ko"
	default:
		return code
	}
}

// translateDLX 调用 DLX（DeepL 兼容）自托管接口。
func translateDLX(ctx context.Context, cfg Config, text string) (string, error) {
	base := strings.TrimRight(cfg.URL, "/")
	if base == "" {
		base = "http://localhost:1188"
	}
	body, _ := json.Marshal(map[string]string{
		"text":        text,
		"source_lang": "auto",
		"target_lang": dlxLang(cfg.Target),
	})
	resp, err := httpPost(ctx, base+"/translate", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析 DLX 响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK || out.Code != http.StatusOK {
		return "", fmt.Errorf("DLX 翻译失败 (HTTP %d)", resp.StatusCode)
	}
	if strings.TrimSpace(out.Data) == "" {
		return "", errors.New("DLX 返回空译文")
	}
	return out.Data, nil
}

// translateBing 调用必应免费翻译接口（免密钥）。
func translateBing(ctx context.Context, cfg Config, text string) (string, error) {
	const endpoint = "https://api-edge.cognitive.microsofttranslator.com/translate"
	return translateBingEndpoint(ctx, cfg, text, endpoint)
}

// translateBingEndpoint 将文本提交到指定的必应兼容端点。
func translateBingEndpoint(ctx context.Context, cfg Config, text, endpoint string) (string, error) {
	body, _ := json.Marshal([]map[string]string{{"Text": text}})
	reqURL := fmt.Sprintf("%s?api-version=3.0&to=%s", endpoint, bingLang(cfg.Target))
	resp, err := httpPost(ctx, reqURL, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("必应翻译失败 (HTTP %d)", resp.StatusCode)
	}
	var out []struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析必应响应失败: %w", err)
	}
	if len(out) == 0 || len(out[0].Translations) == 0 || strings.TrimSpace(out[0].Translations[0].Text) == "" {
		return "", errors.New("必应返回空译文")
	}
	return out[0].Translations[0].Text, nil
}

// translateGoogle 调用 Google 网页翻译接口（client=gtx，免密钥）。
// cfg.URL 可指定镜像/自建代理地址，留空使用官方地址。
func translateGoogle(ctx context.Context, cfg Config, text string) (string, error) {
	endpoint := strings.TrimRight(cfg.URL, "/")
	if endpoint == "" {
		endpoint = "https://translate.googleapis.com"
	}
	return translateGoogleEndpoint(ctx, cfg, text, endpoint)
}

// translateGoogleEndpoint 将文本提交到指定的 Google 翻译兼容端点。
func translateGoogleEndpoint(ctx context.Context, cfg Config, text, endpoint string) (string, error) {
	params := url.Values{
		"client": {"gtx"},
		"sl":     {"auto"},
		"tl":     {googleLang(cfg.Target)},
		"dt":     {"t"},
		"q":      {text},
	}
	reqURL := fmt.Sprintf("%s/translate_a/single?%s", strings.TrimRight(endpoint, "/"), params.Encode())
	resp, err := httpPost(ctx, reqURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google 翻译失败 (HTTP %d)", resp.StatusCode)
	}
	var out []any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析 Google 响应失败: %w", err)
	}
	var parts []string
	collectGoogleSegments(out, &parts)
	translated := strings.TrimSpace(strings.Join(parts, ""))
	if translated == "" {
		return "", errors.New("Google 返回空译文")
	}
	return translated, nil
}

// collectGoogleSegments 递归收集 gtx 响应的翻译片段（out[0][i][0]）。
func collectGoogleSegments(v any, parts *[]string) {
	list, ok := v.([]any)
	if !ok {
		return
	}
	for _, item := range list {
		if seg, ok := item.([]any); ok && len(seg) > 0 {
			if s, ok := seg[0].(string); ok {
				*parts = append(*parts, s)
				continue
			}
		}
		collectGoogleSegments(item, parts)
	}
}

// translateOpenAI 调用 OpenAI 兼容接口（/chat/completions）。
func translateOpenAI(ctx context.Context, cfg Config, text string) (string, error) {
	base := strings.TrimRight(cfg.URL, "/")
	if base == "" {
		return "", errors.New("未配置 OpenAI 兼容接口地址")
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	prompt := fmt.Sprintf(
		"你是一名翻译。请把下面的软件更新日志翻译成%s。只输出译文本身，不要加任何解释、注释或引号，保留原有的 Markdown 格式与换行结构：\n\n%s",
		languageName(cfg.Target), text)

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"temperature": 0.2,
		"messages": []map[string]string{
			{"role": "system", "content": "你是专业翻译。输出只包含译文，不得包含任何额外说明。"},
			{"role": "user", "content": prompt},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI 兼容接口返回 HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("解析 OpenAI 响应失败: %w", err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", errors.New("OpenAI 返回空译文")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func httpPost(ctx context.Context, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
