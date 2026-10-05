package film

import (
	"strings"
	"testing"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

func TestInitScript(t *testing.T) {
	js := InitScript(&scenario.Scenario{Locale: "fr", Hide: []string{".dev-overlay", "#toast"}})
	if want := `({"locale":"fr","hide":[".dev-overlay","#toast"]});`; !strings.HasSuffix(strings.TrimSpace(js), want) {
		t.Errorf("init script must call the overlay with %s, got tail %q", want, js[len(js)-80:])
	}
	if want := `({"locale":"","hide":[]});`; !strings.HasSuffix(strings.TrimSpace(InitScript(&scenario.Scenario{})), want) {
		t.Errorf("an empty scenario must pass %s", want)
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
