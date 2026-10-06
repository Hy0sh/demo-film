package film

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

func TestFilter(t *testing.T) {
	got := Filter(1440, 900, nil, []Window{{0, 3.5}, {3.5, 9.25}})
	want := "[0:v]pad=1440:1050:0:0:color=0x11161b[v0]" +
		";[v0][1:v]overlay=0:900:enable='between(t,0.00,3.50)'[v1]" +
		";[v1][2:v]overlay=0:900:enable='between(t,3.50,9.25)'[v2]"
	if got != want {
		t.Errorf("Filter =\n%s\nwant\n%s", got, want)
	}
}

func TestFilterWithCuts(t *testing.T) {
	got := Filter(1440, 900, []Window{{0, 4}, {64, 70}}, []Window{{0, 9}})
	want := "[0:v]split=2[k0][k1]" +
		";[k0]trim=start=0.000:end=4.000,setpts=PTS-STARTPTS[s0]" +
		";[k1]trim=start=64.000:end=70.000,setpts=PTS-STARTPTS[s1]" +
		";[s0][s1]concat=n=2[cut]" +
		";[cut]pad=1440:1050:0:0:color=0x11161b[v0]" +
		";[v0][1:v]overlay=0:900:enable='between(t,0.00,9.00)'[v1]"
	if got != want {
		t.Errorf("Filter =\n%s\nwant\n%s", got, want)
	}
}

func TestChapters(t *testing.T) {
	got := Chapters("Tour", []Chapter{
		{N: 1, At: 0, Caption: "Open", Expect: "home"},
		{N: 2, Check: "E3", At: 75 * time.Second, Caption: "a | b\nc", Expect: "x"},
	})
	for _, want := range []string{"# Tour", "| Step | Check | Time | Caption | Expect |", "| 1 |  | 0:00 | Open | home |", `| 2 | E3 | 1:15 | a \| b c | x |`} {
		if !strings.Contains(got, want) {
			t.Errorf("chapters lack %q:\n%s", want, got)
		}
	}
}

func TestWatermarkFilter(t *testing.T) {
	for pos, want := range map[string]string{
		"top-left":     ";[3:v]format=rgba,colorchannelmixer=aa=0.60[wm];[v2][wm]overlay=24:24[v3]",
		"bottom-right": ";[3:v]format=rgba,colorchannelmixer=aa=0.60[wm];[v2][wm]overlay=W-w-24:900-h-24[v3]",
	} {
		if got := WatermarkFilter(3, 2, pos, 0.6, 900); got != want {
			t.Errorf("%s: got\n%s\nwant\n%s", pos, got, want)
		}
	}
}

func TestJoinChapters(t *testing.T) {
	a := Chapters("Agent", []Chapter{{N: 1, At: 0, Caption: "Log in", Expect: "home"}, {N: 2, At: 59 * time.Second, Caption: "a | b", Expect: "x"}})
	b := Chapters("Citizen", []Chapter{{N: 1, Check: "E|3", At: 5 * time.Second, Caption: "Sign up", Expect: "y"}})
	var pieces []piece
	for _, md := range []string{a, b} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "chapters.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
		p := piece{dir: dir}
		for _, l := range strings.Split(md, "\n") {
			if strings.HasPrefix(l, "# ") {
				p.title = strings.TrimPrefix(l, "# ")
			} else if chapterRow.MatchString(l) {
				p.rows = append(p.rows, l)
			}
		}
		pieces = append(pieces, p)
	}
	pieces[0].duration = 90 * time.Second
	got := joinChapters(pieces, 2*time.Second)
	for _, want := range []string{"## Agent", "| 1 |  | 0:02 | Log in |", `| 2 |  | 1:01 | a \| b |`, "## Citizen", `| 1 | E\|3 | 1:39 | Sign up |`} {
		if !strings.Contains(got, want) {
			t.Errorf("joined chapters lack %q:\n%s", want, got)
		}
	}
}

func TestCardTime(t *testing.T) {
	if got := CardTime(1); got != 1500*time.Millisecond {
		t.Errorf("CardTime(1) = %s", got)
	}
	if got := CardTime(2); got != minShown {
		t.Errorf("CardTime(2) = %s, want the floor %s", got, minShown)
	}
}

func TestBandHTML(t *testing.T) {
	l := scenario.Labels{Step: "Étape", See: "Tu dois voir :", Check: "vérifie"}
	st := scenario.Step{Caption: "Click <b>", Check: "E3", Expect: "a & b"}
	a := bandHTML(l, 2, 5, st, false)
	for _, want := range []string{"Étape 2/5 · vérifie E3", "Click &lt;b&gt;"} {
		if !strings.Contains(a, want) {
			t.Errorf("state A lacks %q", want)
		}
	}
	if strings.Contains(a, "Tu dois voir") {
		t.Error("state A must not show the expectation")
	}
	if b := bandHTML(l, 2, 5, st, true); !strings.Contains(b, "Tu dois voir :</b> a &amp; b") {
		t.Errorf("state B lacks the expectation: %s", b)
	}
	if strings.Contains(bandHTML(l, 1, 5, scenario.Step{Caption: "c"}, false), "vérifie") {
		t.Error("no check, no check label")
	}
}
