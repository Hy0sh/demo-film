package film

import (
	"strings"
	"testing"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

func TestInitScript(t *testing.T) {
	s := &scenario.Scenario{Locale: "fr", Hide: []string{".dev-overlay", "#toast"}}
	js := InitScript(s)
	for _, want := range []string{`localStorage.setItem("i18nextLng", "fr")`, `.dev-overlay{display:none!important}`, `__demo_cursor`} {
		if !strings.Contains(js, want) {
			t.Errorf("init script lacks %q", want)
		}
	}
	// The locale is set before any overlay code.
	if strings.Index(js, "i18nextLng") > strings.Index(js, "__demo_cursor") {
		t.Error("the locale must be set first")
	}
	if strings.Contains(InitScript(&scenario.Scenario{}), "i18nextLng") {
		t.Error("no locale, no localStorage write")
	}
}

func TestPickOption(t *testing.T) {
	opts := []string{"Red", "Dark Blue", "Blue"}
	if got, _ := pickOption(opts, "Blue"); got != "Blue" {
		t.Errorf("exact match must win, got %q", got)
	}
	if got, _ := pickOption(opts, "dark"); got != "Dark Blue" {
		t.Errorf("substring fallback, got %q", got)
	}
	if _, ok := pickOption(opts, "Green"); ok {
		t.Error("no match must report false")
	}
}
