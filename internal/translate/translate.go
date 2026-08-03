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
	Engine string // dlx / bing / google / openai / youdao
	URL    string // dlx 服务地址 或 openai base_url
	APIKey string // openai 专用
	Model  string // openai 专用
	Target string // 目标语言代码，如 ZH / zh-Hans / en
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
	case "youdao":
		return translateYoudao(ctx, cfg, text)
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

// youdaoLang 将通用目标代码转为有道识别的语言代码。
func youdaoLang(code string) string {
	switch normalizeLang(code) {
	case LangZhHant:
		return "zh-CHT"
	case LangZhHans:
		return "zh-CHS"
	case LangEn:
		return "en"
	case LangJa:
		return "ja"
	case LangKo:
		return "ko"
	default:
		return strings.ToLower(code)
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
// endpoint 可填域名、完整路径（含 /translate_a/single）或镜像地址，均自动归一化。
func translateGoogleEndpoint(ctx context.Context, cfg Config, text, endpoint string) (string, error) {
	base := strings.TrimRight(endpoint, "/")
	if strings.Contains(base, "/translate_a/single") {
		base = strings.TrimSuffix(base, "/translate_a/single")
	}
	if !strings.HasPrefix(base, "http") {
		base = "https://" + base
	}
	params := url.Values{
		"client": {"gtx"},
		"sl":     {"auto"},
		"tl":     {googleLang(cfg.Target)},
		"dt":     {"t"},
		"q":      {text},
	}
	reqURL := fmt.Sprintf("%s/translate_a/single?%s", base, params.Encode())
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

// youdaoMaxChunk 有道单次请求建议的最大字符数。
// 实测约 700 字符内安全、超限返回 411，取 400 保守以避免截断。
const youdaoMaxChunk = 400

// translateYoudao 调用有道免费翻译接口（官方公开演示接口，免密钥）。
func translateYoudao(ctx context.Context, cfg Config, text string) (string, error) {
	const endpoint = "https://aidemo.youdao.com/trans"
	return translateYoudaoEndpoint(ctx, cfg, text, endpoint)
}

// translateYoudaoEndpoint 将文本提交到指定的有道兼容端点。
// 文本较长时按 youdaoMaxChunk 分段、逐段翻译后拼接，以避开接口长度限制。
func translateYoudaoEndpoint(ctx context.Context, cfg Config, text, endpoint string) (string, error) {
	chunks := splitChunks(text, youdaoMaxChunk)
	results := make([]string, len(chunks))
	for i, ch := range chunks {
		res, err := youdaoOne(ctx, cfg, ch, endpoint)
		if err != nil {
			return "", fmt.Errorf("有道翻译第 %d 段失败: %w", i+1, err)
		}
		results[i] = res
	}
	return strings.Join(results, ""), nil
}

// youdaoOne 提交单个分段的文本并解析译文。
func youdaoOne(ctx context.Context, cfg Config, text, endpoint string) (string, error) {
	params := url.Values{
		"q":    {text},
		"from": {"auto"},
		"to":   {youdaoLang(cfg.Target)},
	}
	reqURL := endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		ErrorCode   string   `json:"errorCode"`
		Translation []string `json:"translation"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析有道响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK || out.ErrorCode != "0" {
		return "", fmt.Errorf("有道翻译失败 (HTTP %d, errorCode %s)", resp.StatusCode, out.ErrorCode)
	}
	if len(out.Translation) == 0 || strings.TrimSpace(out.Translation[0]) == "" {
		return "", errors.New("有道返回空译文")
	}
	return out.Translation[0], nil
}

// splitChunks 将文本按行切分为每段不超过 max 个字符（rune）的块。
// 保留原换行符，确保拼接后行结构不变；单行超长时按字符硬切。
func splitChunks(text string, max int) []string {
	lines := strings.SplitAfter(text, "\n")
	var chunks []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			chunks = append(chunks, string(cur))
			cur = nil
		}
	}
	for _, ln := range lines {
		runes := []rune(ln)
		for len(runes) > 0 {
			take := max - len(cur)
			if take > len(runes) {
				take = len(runes)
			}
			if take <= 0 {
				flush()
				continue
			}
			cur = append(cur, runes[:take]...)
			runes = runes[take:]
			if len(cur) >= max {
				flush()
			}
		}
	}
	flush()
	if len(chunks) == 0 {
		return []string{text}
	}
	return chunks
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
