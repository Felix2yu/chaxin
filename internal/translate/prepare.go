package translate

import (
	"context"
	"strings"
)

// extractThreshold 目标语言行文本占比达到该值即判定日志已包含目标语言，直接提取。
const extractThreshold = 0.3

// Result Prepare 的处理结果。
type Result struct {
	Text       string // 最终用于展示/通知的文本（提取或翻译后的）
	Translated bool   // true=经过翻译；false=原样/提取
	Extracted  bool   // true=从多语言日志中提取了目标语言段落
}

// Prepare 处理一段更新日志：
//  1. 空文本直接返回；
//  2. 若日志已包含足够比例的目标语言内容，直接提取目标语言段落（不翻译）；
//  3. 否则调用配置的翻译引擎整段翻译。
//
// cfg.Engine 为 off 或未配置时，若检测到已是目标语言则返回原文本，否则返回 ErrNotConfigured。
func Prepare(ctx context.Context, cfg Config, body string) (Result, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Result{Text: body}, nil
	}

	target := normalizeLang(cfg.Target)

	// 按行切分并逐行检测语言
	lines := strings.Split(body, "\n")
	var targetLines []string
	targetLen := 0
	totalLen := 0
	for _, ln := range lines {
		totalLen += len([]rune(ln))
		if detectLine(ln) == target || (target == LangZhHans && detectLine(ln) == LangZhHant) {
			targetLines = append(targetLines, ln)
			targetLen += len([]rune(ln))
		}
	}

	// 已包含足够目标语言内容：直接提取
	if totalLen > 0 && float64(targetLen)/float64(totalLen) >= extractThreshold {
		text := strings.TrimSpace(strings.Join(targetLines, "\n"))
		if text == "" {
			text = body
		}
		return Result{Text: text, Extracted: targetLen < totalLen}, nil
	}

	// 主体是外文，需要翻译
	if cfg.Engine == "" || cfg.Engine == "off" {
		return Result{}, ErrNotConfigured
	}
	translated, err := Translate(ctx, cfg, body)
	if err != nil {
		return Result{}, err
	}
	return Result{Text: translated, Translated: true}, nil
}
