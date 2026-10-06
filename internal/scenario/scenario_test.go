package scenario

import (
	_ "embed"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/valid.yaml
var valid string

func problems(t *testing.T, yaml string) string {
	t.Helper()
	s, err := Parse([]byte(yaml))
	if err != nil {
		return "parse: " + err.Error()
	}
	return strings.Join(Validate(s), "\n")
}

func TestValidScenario(t *testing.T) {
	if got := problems(t, valid); got != "" {
		t.Fatalf("valid scenario rejected: %s", got)
	}
	s, _ := Parse([]byte(valid))
	if s.Viewport.Width != 1440 || s.Viewport.Height != 900 || s.Timeout != 15 || s.Labels.See != "You should see:" {
		t.Errorf("defaults not applied: %+v", s)
	}
}

func TestRules(t *testing.T) {
	cases := []struct {
		name, from, to, want string
	}{
		{"empty title", "title: Tour", "title: ''", "title is empty"},
		{"empty base_url", "base_url: http://localhost:3000", "base_url: ''", "base_url is empty"},
		{"relative base_url", "base_url: http://localhost:3000", "base_url: localhost", "not an absolute URL"},
		{"odd viewport", "title: Tour", "title: Tour\nviewport: {width: 1441, height: 900}", "must be even"},
		{"speed too slow", "title: Tour", "title: Tour\nspeed: 0.1", "speed must be between"},
		{"speed too fast", "title: Tour", "title: Tour\nspeed: 5", "speed must be between"},
		{"empty caption", "caption: Open the app", "caption: ''", "caption is empty"},
		{"missing expect", "    expect: the home page\n", "", "expect is empty"},
		{"check without see", "    see: [Home]\n", "", "needs a non-empty see"},
		{"no action", "    do:\n      - open: /\n      - wait: Home\n", "    do: []\n", "do has no action"},
		{"same-origin open after step 1", "      - menu: [Settings, Profile]", "      - open: /settings", "reloads the app"},
		{"unknown verb", "      - hover: Help", "      - drag: Help", "unknown action"},
		{"two verbs", "      - hover: Help", "      - hover: Help\n        press: Enter", "exactly one verb"},
		{"unknown key", "title: Tour", "title: Tour\nunknown: 1", "field unknown not found"},
		{"unknown key in click", "{role: button, name: Edit}", "{role: button, label: Edit}", "field label not found"},
		{"within outside its verbs", "      - press: Enter", "      - press: Enter\n        within: dialog", "within is allowed on"},
		{"within other than dialog", "within: dialog", "within: drawer", "only dialog"},
		{"terminal and base_url", "title: Tour", "title: Tour\nterminal: {shell: sh}", "exclude each other"},
		{"empty type", "      - press: Enter", "      - type: ''", "the value is empty"},
		{"watermark without source", "title: Tour", "title: Tour\nwatermark: {position: top-left}", "exactly one of image or text"},
		{"watermark with both", "title: Tour", "title: Tour\nwatermark: {text: a, image: b.png}", "exactly one of image or text"},
		{"watermark image missing", "title: Tour", "title: Tour\nwatermark: {image: /nowhere/logo.png}", "no such file"},
		{"watermark position", "title: Tour", "title: Tour\nwatermark: {text: a, position: middle}", "is not top-left"},
		{"watermark opacity", "title: Tour", "title: Tour\nwatermark: {text: a, opacity: 2}", "opacity must be between"},
		{"cut outside wait", "      - press: Enter", "      - press: Enter\n        cut: true", "allowed on wait only"},
		{"timeout outside wait", "      - press: Enter", "      - press: Enter\n        timeout: 60", "allowed on wait only"},
		{"negative wait timeout", "      - wait: Home", "      - wait: Home\n        timeout: -1", "timeout must be positive"},
		{"popup without url", "{click: Docs, url_contains: /docs}", "{click: Docs}", "popup needs"},
		{"menu with one label", "[Settings, Profile]", "[Settings]", "menu needs exactly"},
		{"wait on two conditions", "      - wait: Home", "      - wait: {gone: Toast, enabled: Next}", "wait is a text, {gone: text} or {enabled: text}"},
		{"wait on an unknown condition", "      - wait: Home", "      - wait: {hidden: Toast}", "field hidden not found"},
		{"open in a terminal demo", "base_url: http://localhost:3000", "terminal: {shell: sh}", "open would restart it"},
		{"nth on a row click", "{row: Ada, button: Edit}", "{row: Ada, button: Edit, nth: 1}", "nth goes with text or {role, name}"},
		{"text and role together", "{role: button, name: Edit}", "{role: button, name: Edit, text: Edit}", "click is a text"},
		{"last click is a forward button", "      - confirm: Discard", "      - click: Suivant", "forward button"},
		{"last role click is a forward button", "      - confirm: Discard", "      - click: {role: button, name: Next}", "forward button"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(valid, c.from) {
				t.Fatalf("test bug: %q not in the base scenario", c.from)
			}
			got := problems(t, strings.Replace(valid, c.from, c.to, 1))
			if !strings.Contains(got, c.want) {
				t.Errorf("want a problem containing %q, got %q", c.want, got)
			}
		})
	}
}

func TestCutWait(t *testing.T) {
	s, err := Parse([]byte(strings.Replace(valid, "      - wait: Home", "      - wait: Home\n        cut: true\n        timeout: 600", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if errs := Validate(s); len(errs) > 0 {
		t.Fatalf("a cut wait must pass: %v", errs)
	}
	if w := s.Steps[0].Do[1]; !w.Cut || w.Timeout != 600 || w.Text != "Home" {
		t.Errorf("cut wait decoded as %+v", w)
	}
}

func TestWaitConditions(t *testing.T) {
	y := strings.Replace(valid, "      - wait: Home", "      - wait: {gone: Saving…}\n        within: dialog\n        cut: true\n        timeout: 60\n      - wait: {enabled: Next}\n      - wait: Home", 1)
	s, err := Parse([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if errs := Validate(s); len(errs) > 0 {
		t.Fatalf("waits on conditions must pass: %v", errs)
	}
	do := s.Steps[0].Do
	if g, e := do[1], do[2]; g.Until != Gone || g.Text != "Saving…" || !g.Cut || g.Within != "dialog" || e.Until != Enabled || e.Text != "Next" {
		t.Errorf("decoded as %+v and %+v", g, e)
	}
	if got := do[1].String(); got != `wait until "Saving…" is gone within dialog (cut)` {
		t.Errorf("String() = %s", got)
	}
}

func TestClickNth(t *testing.T) {
	y := strings.Replace(valid, "      - hover: Help", "      - click: {text: Slot, nth: 1}\n      - click: {role: button, name: Actions, nth: -1}\n      - hover: Help", 1)
	s, err := Parse([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if errs := Validate(s); len(errs) > 0 {
		t.Fatalf("click with nth must pass: %v", errs)
	}
	var got []string
	for _, a := range s.Steps[1].Do {
		if a.Match != nil {
			got = append(got, a.String())
		}
	}
	want := []string{`click "Slot" (match #1)`, `click button "Actions" (match #-1)`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOpenRules(t *testing.T) {
	multi := func(open string) string {
		return strings.Replace(valid, "      - menu: [Settings, Profile]", "      - open: "+open+"\n      - menu: [Settings, Profile]", 1)
	}
	if got := problems(t, multi("http://localhost:8025/inbox")); got != "" {
		t.Errorf("open on another origin must pass: %s", got)
	}
	if got := problems(t, multi("http://localhost:3000/x")); !strings.Contains(got, "reloads the app") {
		t.Errorf("absolute same-origin open must fail: %q", got)
	}
	// Leaving for a mail catcher and coming back is allowed.
	back := strings.Replace(valid, "      - menu: [Settings, Profile]", "      - open: http://localhost:8025\n      - open: /\n      - menu: [Settings, Profile]", 1)
	if got := problems(t, back); got != "" {
		t.Errorf("open back to the app from another origin must pass: %s", got)
	}
	// Step 1 may open any number of times.
	two := strings.Replace(valid, "      - wait: Home", "      - open: /login\n      - wait: Home", 1)
	if got := problems(t, two); got != "" {
		t.Errorf("several opens in step 1 must pass: %s", got)
	}
}

func TestForwardButtonOnlyFlagsTheLastAction(t *testing.T) {
	ok := strings.Replace(valid, "      - hover: Help", "      - click: Next\n      - hover: Help", 1)
	if got := problems(t, ok); got != "" {
		t.Errorf("a Next click in the middle must pass: %s", got)
	}
	// A longer name is not a bare forward button.
	ok = strings.Replace(valid, "      - confirm: Discard", "      - click: Next of kin", 1)
	if got := problems(t, ok); got != "" {
		t.Errorf("only the bare name is flagged: %s", got)
	}
}

func TestAllProblemsAreListed(t *testing.T) {
	got := problems(t, "title: ''\nbase_url: ''\nsteps: []\n")
	for _, w := range []string{"title is empty", "base_url is empty", "steps is empty"} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in %q", w, got)
		}
	}
}

func TestFieldRankAndLabel(t *testing.T) {
	s, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	do := s.Steps[1].Do
	if do[2].Field.Label != "Name" || do[3].Field.Rank != 2 {
		t.Errorf("fields: %+v %+v", do[2].Field, do[3].Field)
	}
	if do[4].Nth == nil || *do[4].Nth != -2 || do[5].Button != "Edit" || do[9].Within != "dialog" {
		t.Errorf("click shapes: %+v %+v %+v", do[4], do[5], do[9])
	}
}

func TestPathsAreRelativeToTheScenario(t *testing.T) {
	dir := filepath.Join("testdata", "relative")
	s, err := Load(filepath.Join(dir, "demo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Terminal.Cwd != filepath.Join(dir, "work") || s.Watermark.Image != filepath.Join(dir, "logo.png") {
		t.Errorf("cwd %q, image %q: want both under %s", s.Terminal.Cwd, s.Watermark.Image, dir)
	}
}

func TestExampleScenarioIsValid(t *testing.T) {
	if _, err := Load("../../examples/shop.yaml"); err != nil {
		t.Fatal(err)
	}
}
