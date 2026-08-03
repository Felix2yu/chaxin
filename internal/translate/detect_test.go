package translate

import "testing"

func TestDetectLine(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"", ""},
		{"=== !!! --- ...", ""},
		{"This is an English release note.", LangEn},
		{"Fix the bug and improve performance.", LangEn},
		{"新增了强大的新功能。", LangZhHans},
		{"修复了许多问题。", LangZhHans},
		{"我们支持简体中文。", LangZhHans},
		{"我們支持繁體中文。", LangZhHant},
		{"這是繁體中文的更新日誌。", LangZhHant},
		{"バグを修正しました。", LangJa},
		{"新しい機能を追加しました。", LangJa},
		{"새로운 기능이 추가되었습니다.", LangKo},
		{"バグ修正とカタカナテスト。", LangJa},
		{"### 日本語", LangJa},
		{"#### 追加", LangJa},
		{"#### 変更", LangJa},
		{"#### 修正", LangJa},
		{"#### 改善", LangJa},
		{"#### 新規", LangJa},
		{"### 简体中文", LangZhHans},
		{"#### 新增", LangZhHans},
		{"#### 改进", LangZhHans},
	}
	for _, c := range cases {
		if got := detectLine(c.line); got != c.want {
			t.Errorf("detectLine(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestNormalizeLang(t *testing.T) {
	cases := map[string]string{
		"zh-Hans": LangZhHans,
		"zh-Hant": LangZhHant,
		"ZH":      LangZhHans,
		"en":      LangEn,
		"ja":      LangJa,
		"ko":      LangKo,
		"zh":      LangZhHans,
		"zhtw":    LangZhHant,
	}
	for in, want := range cases {
		if got := normalizeLang(in); got != want {
			t.Errorf("normalizeLang(%q) = %q, want %q", in, got, want)
		}
	}
}
