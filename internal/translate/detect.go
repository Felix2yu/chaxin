package translate

import (
	"strings"
)

// Lang 目标语言类别（前端下拉使用），检测与提取均按类别匹配。
const (
	LangZhHans = "zh-Hans" // 简体中文
	LangZhHant = "zh-Hant" // 繁体中文
	LangEn     = "en"
	LangJa     = "ja"
	LangKo     = "ko"
)

// 简体/繁体独有字集合（用于区分简繁，覆盖常见用字即可）。
const (
	simpChars = "这国说门时发见对长东还个与后经红线过进让认设实话条图问现样应于云张们体关观风车华万电间单乐处机权热马鱼书买卖阳阴队难虽显爱"
	tradChars = "這國說門時發見對長東還隻個與後經紅線過進讓認設實話條圖問現樣應於雲張們體關觀風車華萬電間單樂處機權熱馬魚書買賣陽陰隊難雖顯愛"
)

// jaHeadingWords 日文更新日志常用的纯汉字标题词（无假名时据此识别）。
var jaHeadingWords = []string{"日本語", "追加", "変更", "修正", "改善", "新規", "廃止", "削除"}

// isJaHeading 判断一行是否为日文标题行：去除 markdown 标题符号后，
// 剩余内容恰好为日文常用标题词。用于识别纯汉字、无假名的日文标题。
func isJaHeading(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	// 允许 markdown 标题前缀（#、-、* 及数字序号）
	trimmed := strings.TrimLeft(s, "#*-. ")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return false
	}
	for _, w := range jaHeadingWords {
		if trimmed == w {
			return true
		}
	}
	return false
}

// normalizeLang 将用户选择/配置的语言归一到检测类别。
// 简体与繁体归为同一"中文"类别用于检测提取，保留原始代码用于翻译引擎。
func normalizeLang(code string) string {
	switch strings.ToLower(strings.ReplaceAll(code, "-", "")) {
	case "zh", "zhhans", "zhcn":
		return LangZhHans
	case "zhhant", "zhtw", "zhhk":
		return LangZhHant
	case "en":
		return LangEn
	case "ja", "jp":
		return LangJa
	case "ko", "kr":
		return LangKo
	default:
		return strings.ToLower(code)
	}
}

// detectLine 判定单行的语言类别（字符启发式）。
// 返回 "" 表示该行无可识别语言特征（空行/纯标点/代码）。
func detectLine(line string) string {
	var (
		hasCJK    bool
		hasHira   bool
		hasKata   bool
		hasHangul bool
		hasLatin  bool
	)
	simp, trad := 0, 0

	for _, r := range line {
		switch {
		case r >= '\u3040' && r <= '\u309f': // 平假名
			hasHira = true
		case r >= '\u30a0' && r <= '\u30ff': // 片假名
			hasKata = true
		case r >= '\uac00' && r <= '\ud7af': // 谚文
			hasHangul = true
		case r >= '\u4e00' && r <= '\u9fff': // CJK 统一表意文字
			hasCJK = true
			if strings.ContainsRune(simpChars, r) {
				simp++
			}
			if strings.ContainsRune(tradChars, r) {
				trad++
			}
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLatin = true
		}
	}

	switch {
	case hasHira || hasKata:
		return LangJa
	case hasHangul:
		return LangKo
	case hasCJK:
		// 纯汉字日文标题（如 日本語/追加/変更）无假名，需用词表识别
		if isJaHeading(line) {
			return LangJa
		}
		// 简繁判定：以特有字计数为准；无证据时归为简体
		if trad > simp && simp == 0 {
			return LangZhHant
		}
		return LangZhHans
	case hasLatin:
		return LangEn
	default:
		return ""
	}
}


