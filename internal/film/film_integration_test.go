package film_test

import (
	"fmt"
	pngdec "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Hy0sh/demo-film/internal/film"
	"github.com/Hy0sh/demo-film/internal/scenario"
)

func ffprobe(t *testing.T, file, entries string) string {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", entries, "-of", "csv=p=0", file).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// blankFirstFrame reports whether the page area of the first frame is a
// single colour.
func blankFirstFrame(t *testing.T, mp4 string, w, h int) bool {
	t.Helper()
	png := filepath.Join(t.TempDir(), "first.png")
	crop := fmt.Sprintf("crop=%d:%d:0:0", w, h)
	if out, err := exec.Command("ffmpeg", "-v", "error", "-i", mp4, "-frames:v", "1", "-vf", crop, png).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, out)
	}
	f, err := os.Open(png)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := pngdec.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	first := img.At(b.Min.X, b.Min.Y)
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			if img.At(x, y) != first {
				return false
			}
		}
	}
	return true
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
	if blankFirstFrame(t, mp4, s.Viewport.Width, s.Viewport.Height) {
		t.Error("the video starts on a blank page: the pre-roll must cut the app's loading")
	}
	if height := ffprobe(t, mp4, "stream=height"); height != strconv.Itoa(s.Viewport.Height+film.BandHeight) {
		t.Errorf("height = %s, want %d", height, s.Viewport.Height+film.BandHeight)
	}
	if pix := ffprobe(t, mp4, "stream=pix_fmt"); pix != "yuv420p" {
		t.Errorf("pix_fmt = %s", pix)
	}
	// Captions set the floor; each action may add up to one gesture pace.
	var floor, gestures float64
	for _, st := range s.Steps {
		floor += (film.ReadTime(st.Caption, s.Speed) + film.HoldTime(st.Expect, s.Speed)).Seconds()
		gestures += float64(len(st.Do)) * film.ActionPace(s.Speed).Seconds()
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", mp4).Output()
	if err != nil {
		t.Fatal(err)
	}
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if ceil := floor + gestures + 30; dur < floor || dur > ceil {
		t.Errorf("duration %.1fs, want between %.1fs and %.1fs", dur, floor, ceil)
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

func TestFilmCutsALongWait(t *testing.T) {
	requireTools(t)
	app, _ := newApps(t)
	s, err := scenario.Parse([]byte(fmt.Sprintf(`
title: Export
base_url: %s
steps:
  - caption: Open the shop
    do:
      - open: /
    see: [Home]
    expect: the home page
  - caption: Export the catalogue
    do:
      - click: Start export
      - wait: Export done
        cut: true
        timeout: 30
    see: [Export done]
    expect: the export is done
`, app.URL)))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := film.Film(s, dir); err != nil {
		t.Fatal(err)
	}
	var floor float64
	for _, st := range s.Steps {
		floor += (film.ReadTime(st.Caption, s.Speed) + film.HoldTime(st.Expect, s.Speed)).Seconds()
	}
	floor += 2 * film.ActionPace(s.Speed).Seconds()
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", filepath.Join(dir, "demo.mp4")).Output()
	if err != nil {
		t.Fatal(err)
	}
	// The 12 s export is gone; the badge and gestures stay well under it.
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if dur < floor || dur > floor+6 {
		t.Errorf("duration %.1fs, want between %.1fs and %.1fs: the wait was not cut", dur, floor, floor+6)
	}
}

func TestFilmWithAWatermark(t *testing.T) {
	requireTools(t)
	app, _ := newApps(t)
	s, err := scenario.Parse([]byte(fmt.Sprintf(`
title: Signed
base_url: %s
speed: 4
watermark: {text: "© Demo", position: top-right}
steps:
  - caption: Open the shop
    do:
      - open: /
    see: [Home]
    expect: the home page
`, app.URL)))
	if err != nil {
		t.Fatal(err)
	}
	if errs := scenario.Validate(s); len(errs) > 0 {
		t.Fatal(errs)
	}
	if err := film.Film(s, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestJoin(t *testing.T) {
	requireTools(t)
	app, _ := newApps(t)
	part := func(title string, viewport int) string {
		s, err := scenario.Parse([]byte(fmt.Sprintf(`
title: %s
base_url: %s
speed: 4
viewport: {width: %d, height: 600}
steps:
  - caption: Open the shop
    do:
      - open: /
    see: [Home]
    expect: the home page
`, title, app.URL, viewport)))
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := film.Film(s, dir); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	a, b := part("Agent side", 800), part("Citizen side", 800)

	out := t.TempDir()
	if err := film.Join([]string{a, b}, out, true); err != nil {
		t.Fatal(err)
	}
	duration := func(dir string) float64 {
		o, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", filepath.Join(dir, "demo.mp4")).Output()
		if err != nil {
			t.Fatal(err)
		}
		d, _ := strconv.ParseFloat(strings.TrimSpace(string(o)), 64)
		return d
	}
	want := duration(a) + duration(b) + 2*film.CardTime(4).Seconds()
	if got := duration(out); got < want-0.5 || got > want+0.5 {
		t.Errorf("joined duration %.2fs, want %.2fs (both films and two cards)", got, want)
	}
	md, err := os.ReadFile(filepath.Join(out, "chapters.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"## Agent side", "## Citizen side"} {
		if !strings.Contains(string(md), w) {
			t.Errorf("chapters.md lacks %q:\n%s", w, md)
		}
	}

	// A part filmed at another viewport is refused, by name.
	c := part("Odd one", 640)
	if err := film.Join([]string{a, c}, t.TempDir(), true); err == nil || !strings.Contains(err.Error(), c) {
		t.Errorf("want a refusal naming %s, got %v", c, err)
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
