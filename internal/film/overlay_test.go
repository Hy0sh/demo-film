package film

import (
	"fmt"
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
	// A French option label holds a narrow no-break space before the colon.
	if got, ok := pickOption([]string{"Lieu : Paris"}, "Lieu : Paris"); !ok || got != "Lieu : Paris" {
		t.Errorf("spaces must match any white space, got %q", got)
	}
}

func TestMatch(t *testing.T) {
	if got := match("Save", true); got != "Save" {
		t.Errorf("a text without space stays a text, got %v", got)
	}
	for _, c := range []struct {
		text  string
		exact bool
		want  string
	}{
		{"ex : Natation", true, `^\s*ex\s+:\s+Natation\s*$`},
		{"ex : Natation", true, `^\s*ex\s+:\s+Natation\s*$`},
		{"11h30 / 13h30 (x)", false, `(?i)11h30\s+\/\s+13h30\s+\(x\)`},
	} {
		if got := fmt.Sprint(match(c.text, c.exact)); got != c.want {
			t.Errorf("match(%q, %v) = %s, want %s", c.text, c.exact, got, c.want)
		}
	}
}
