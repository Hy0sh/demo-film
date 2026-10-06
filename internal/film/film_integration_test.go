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
