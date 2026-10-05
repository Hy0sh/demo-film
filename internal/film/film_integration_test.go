package film_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Hy0sh/demo-film/internal/film"
)

func ffprobe(t *testing.T, file, entries string) string {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", entries, "-of", "csv=p=0", file).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestFilmWritesVideoAndChapters(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	dir := t.TempDir()
	if err := film.Film(s, dir); err != nil {
		t.Fatal(err)
	}

	mp4 := filepath.Join(dir, "demo.mp4")
	if height := ffprobe(t, mp4, "stream=height"); height != strconv.Itoa(s.Viewport.Height+film.BandHeight) {
		t.Errorf("height = %s, want %d", height, s.Viewport.Height+film.BandHeight)
	}
	if pix := ffprobe(t, mp4, "stream=pix_fmt"); pix != "yuv420p" {
		t.Errorf("pix_fmt = %s", pix)
	}
	var floor float64
	for _, st := range s.Steps {
		floor += (film.ReadTime(st.Caption) + film.HoldTime(st.Expect)).Seconds()
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", mp4).Output()
	if err != nil {
		t.Fatal(err)
	}
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if dur < floor || dur > floor+30 {
		t.Errorf("duration %.1fs, want between %.1fs and %.1fs", dur, floor, floor+30)
	}

	md, err := os.ReadFile(filepath.Join(dir, "chapters.md"))
	if err != nil {
		t.Fatal(err)
	}
	var rows int
	for _, l := range strings.Split(string(md), "\n") {
		if strings.HasPrefix(l, "| ") && !strings.HasPrefix(l, "| Step") {
			rows++
		}
	}
	if rows != len(s.Steps) {
		t.Errorf("chapters.md has %d rows, want %d:\n%s", rows, len(s.Steps), md)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "demo.mp4" && e.Name() != "chapters.md" {
			t.Errorf("leftover in the output dir: %s", e.Name())
		}
	}
}

func TestFilmAbortsWithoutVideoWhenASeeFails(t *testing.T) {
	requireTools(t)
	app, mail := newApps(t)
	s := scenarioFor(t, app.URL, mail.URL)
	s.Timeout = 1
	s.Steps[0].See = []string{"Nowhere to be seen"}
	dir := t.TempDir()
	if err := film.Film(s, dir); err == nil || !strings.Contains(err.Error(), "step 1") {
		t.Fatalf("want a failure at step 1, got %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a failed film must leave nothing behind, found %d entries", len(entries))
	}
}
