package i18n

import (
	"testing"
)

func TestT_ExactMatch(t *testing.T) {
	SetLang("zh")
	Register("test",
		"en", Dict{"greet": "Hello"},
		"zh", Dict{"greet": "你好"},
	)

	if got := T("test.greet"); got != "你好" {
		t.Errorf("expected 你好, got %s", got)
	}
}

func TestT_FallbackToEnglish(t *testing.T) {
	SetLang("fr")
	Register("test",
		"en", Dict{"hello": "Hello"},
	)

	if got := T("test.hello"); got != "Hello" {
		t.Errorf("expected Hello, got %s", got)
	}
}

func TestT_FallbackToKey(t *testing.T) {
	SetLang("en")
	if got := T("nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("expected key itself, got %s", got)
	}
}

func TestT_WithArgs(t *testing.T) {
	SetLang("en")
	Register("test",
		"en", Dict{"welcome": "Welcome, %s!"},
	)

	if got := T("test.welcome", "Alice"); got != "Welcome, Alice!" {
		t.Errorf("expected Welcome, Alice!, got %s", got)
	}
}

func TestLang(t *testing.T) {
	SetLang("zh")
	if got := Lang(); got != "zh" {
		t.Errorf("expected zh, got %s", got)
	}
}
