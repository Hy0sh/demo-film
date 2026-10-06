package film_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"

	"github.com/Hy0sh/demo-film/internal/film"
	"github.com/Hy0sh/demo-film/internal/scenario"
)

// scenarioFor covers every action of the vocabulary on the test app.
func scenarioFor(t *testing.T, appURL, mailURL string) *scenario.Scenario {
	t.Helper()
	return loadScenario(t, "tour", map[string]string{"App": appURL, "Mail": mailURL})
}

// requireTools skips the test unless a browser and ffmpeg are available.
func requireTools(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test")
	}
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("playwright driver missing (run demo-film install): %v", err)
	}
	defer pw.Stop()
	b, err := pw.Chromium.Launch()
	if err != nil {
		t.Skipf("chromium missing (run demo-film install): %v", err)
	}
	b.Close()
}

func TestRehearseCoversEveryAction(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	if err := film.Rehearse(s, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestRehearseRefusesAPickerThatDropsItsValue(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Steps[2].Do = append(s.Steps[2].Do, scenario.Action{Kind: scenario.Fill, Field: scenario.Field{Label: "Locked"}, Value: "2026-10-06"})
	err := film.Rehearse(s, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), `holds "", not "2026-10-06"`) {
		t.Fatalf("want the dropped value reported, got %v", err)
	}
}

// The context's locale is checked through Intl; the --lang that formats
// native date inputs renders in their shadow DOM, out of reach of see.
func TestLocaleReachesTheBrowser(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Locale = "fr"
	s.Steps = s.Steps[:1]
	s.Steps[0].See = []string{"Month: octobre"}
	if err := film.Rehearse(s, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestFilmATerminal(t *testing.T) {
	requireTools(t)
	if _, err := exec.LookPath("ttyd"); err != nil {
		t.Skip("ttyd not installed")
	}
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "marker.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := loadScenario(t, "terminal", map[string]string{"Cwd": cwd})
	if err := film.Rehearse(s, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := film.Film(s, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo.mp4")); err != nil {
		t.Fatal(err)
	}
}

func TestRehearseNamesHowManyMatchWhenNthIsOutOfRange(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Timeout = 1
	five := 5
	s.Steps[4].Do[0].Match = &five
	err := film.Rehearse(s, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "nth 5 is out of range: 3 visible elements match") {
		t.Fatalf("want the match count, got %v", err)
	}
}

func TestRehearseFailsAtTheRightStep(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Timeout = 1
	s.Steps[2].See = []string{"Submitted: nothing like this"}
	dir := t.TempDir()
	err := film.Rehearse(s, dir)
	if err == nil || !strings.Contains(err.Error(), "step 3") || !strings.Contains(err.Error(), "Submitted: nothing like this") {
		t.Fatalf("want a failure naming step 3 and the text, got %v", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "rehearse-fail-step3.png")); serr != nil {
		t.Errorf("no screenshot: %v", serr)
	}
}

func TestRehearseNamesTheFailingAction(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Timeout = 1
	s.Steps[3].Do[1] = scenario.Action{Kind: scenario.Click, Text: "No such button"}
	err := film.Rehearse(s, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "step 4") || !strings.Contains(err.Error(), `click "No such button"`) {
		t.Fatalf("want step 4 and the action, got %v", err)
	}
	// The miss lists what the screen offers, so the label is fixed from the error.
	if !strings.Contains(err.Error(), `visible on this screen:`) || !strings.Contains(err.Error(), `"Open wizard"`) {
		t.Errorf("want the visible names in the error, got %v", err)
	}
}
