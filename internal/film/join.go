package film

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

//go:embed title_card.html
var titleCardSource string

var titleCard = template.Must(template.New("card").Parse(titleCardSource))

// cardHTML is the title card of a film of that frame height.
func cardHTML(title string, height int) (string, error) {
	var b strings.Builder
	err := titleCard.Execute(&b, struct {
		Title, Background, Accent string
		Height                    int
	}{title, bandBackground, bandAccent, height})
	return b.String(), err
}

// fps is the frame rate join brings every input to, the one Playwright
// records at, so the films go through untouched and the cards match them.
const fps = 25

// cardBase is how long a title card shows at speed 1.
const cardBase = 1500 * time.Millisecond

// CardTime is how long join shows a title card at that speed: like the
// captions, scaled by the speed and never under minShown.
func CardTime(speed float64) time.Duration {
	return max(time.Duration(float64(cardBase)/speed), minShown)
}

// piece is the output of one film, as join reads it back.
type piece struct {
	dir           string
	video         string // absolute: a dir named "-x" is no ffmpeg option
	title         string
	rows          []Chapter // read back from chapters.md, to the second
	width, height int
	codec, pixFmt string
	speed         float64
	duration      time.Duration
}

// Join assembles the outputs of several films, in the order given, into
// outDir: demo.mp4, each film preceded by a title card unless cards is
// false, and chapters.md with one section per film, its times shifted by
// what comes before it. Films that differ in frame size, codec, pixel
// format or speed are refused, naming the film at fault.
func Join(dirs []string, outDir string, cards bool) error {
	var pieces []piece
	for _, d := range dirs {
		if abs(d) == abs(outDir) {
			return fmt.Errorf("%s: the output directory cannot be one of the films", d)
		}
		p, err := readPiece(d)
		if err != nil {
			return err
		}
		if len(pieces) > 0 {
			if err := sameFormat(pieces[0], p); err != nil {
				return err
			}
		}
		pieces = append(pieces, p)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	tmp, done, err := tempDir(outDir)
	if err != nil {
		return err
	}
	defer done() // card PNGs

	card := time.Duration(0)
	var pngs []string
	if cards {
		card = CardTime(pieces[0].speed)
		if pngs, err = renderCards(pieces, tmp); err != nil {
			return err
		}
	}

	var args []string
	var filter strings.Builder
	n := 0
	for i, p := range pieces {
		if cards {
			args = append(args, "-loop", "1", "-framerate", strconv.Itoa(fps), "-t", fmt.Sprintf("%.3f", card.Seconds()), "-i", pngs[i])
			fmt.Fprintf(&filter, "[%d:v]fps=%d,format=yuv420p,setsar=1[p%d];", n, fps, n)
			n++
		}
		args = append(args, "-i", p.video)
		fmt.Fprintf(&filter, "[%d:v]fps=%d,format=yuv420p,setsar=1[p%d];", n, fps, n)
		n++
	}
	for i := range n {
		fmt.Fprintf(&filter, "[p%d]", i)
	}
	fmt.Fprintf(&filter, "concat=n=%d[out]", n)
	args = append(args, "-filter_complex", filter.String(), "-map", "[out]")
	if err := ffmpeg(append(args, encodeArgs(pieces[0].speed, filepath.Join(tmp, "demo.mp4"))...)...); err != nil {
		return err
	}
	md, err := joinChapters(pieces, card)
	if err != nil {
		return err
	}
	return publish(tmp, outDir, md)
}

func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

// readPiece reads a film's output directory: chapters.md for the title and
// the steps, ffprobe on demo.mp4 for the format, the speed and the length.
func readPiece(dir string) (piece, error) {
	p := piece{dir: dir, video: filepath.Join(abs(dir), "demo.mp4")}
	md, err := os.ReadFile(filepath.Join(dir, "chapters.md"))
	if err != nil {
		return p, fmt.Errorf("%s: not the output of a film: %w", dir, err)
	}
	for _, l := range strings.Split(string(md), "\n") {
		switch {
		case strings.HasPrefix(l, "# ") && p.title == "":
			p.title = strings.TrimPrefix(l, "# ")
		default:
			if c, ok := readChapter(l); ok {
				p.rows = append(p.rows, c)
			}
		}
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name,width,height,pix_fmt:format=duration:format_tags="+speedTag,
		"-of", "default=nw=1", p.video).Output()
	if err != nil {
		return p, fmt.Errorf("%s: ffprobe demo.mp4: %w", dir, err)
	}
	for _, l := range strings.Split(string(out), "\n") {
		k, v, _ := strings.Cut(strings.TrimSpace(l), "=")
		switch k {
		case "codec_name":
			p.codec = v
		case "pix_fmt":
			p.pixFmt = v
		case "width":
			p.width, _ = strconv.Atoi(v)
		case "height":
			p.height, _ = strconv.Atoi(v)
		case "duration":
			s, _ := strconv.ParseFloat(v, 64)
			p.duration = time.Duration(s * float64(time.Second))
		case "TAG:" + speedTag:
			p.speed, _ = strconv.ParseFloat(v, 64)
		}
	}
	if p.speed == 0 {
		return p, fmt.Errorf("%s: demo.mp4 carries no speed: it was filmed before join existed, film it again", dir)
	}
	return p, nil
}

// sameFormat refuses a film the first one cannot be joined with.
func sameFormat(first, p piece) error {
	switch {
	case p.width != first.width || p.height != first.height:
		return fmt.Errorf("%s: frame %dx%d, but %s is %dx%d: film every part with the same viewport", p.dir, p.width, p.height, first.dir, first.width, first.height)
	case p.codec != first.codec || p.pixFmt != first.pixFmt:
		return fmt.Errorf("%s: %s %s, but %s is %s %s", p.dir, p.codec, p.pixFmt, first.dir, first.codec, first.pixFmt)
	case p.speed != first.speed:
		return fmt.Errorf("%s: speed %g, but %s is %g: film every part at the same speed", p.dir, p.speed, first.dir, first.speed)
	}
	return nil
}

// renderCards draws each film's title card, at the films' frame size.
func renderCards(pieces []piece, dir string) ([]string, error) {
	pw, browser, err := startBrowser()
	if err != nil {
		return nil, err
	}
	defer pw.Stop()
	defer browser.Close()
	w, h := pieces[0].width, pieces[0].height
	page, err := browser.NewPage(playwright.BrowserNewPageOptions{Viewport: &playwright.Size{Width: w, Height: h}})
	if err != nil {
		return nil, err
	}
	var pngs []string
	for i, p := range pieces {
		card, err := cardHTML(p.title, h)
		if err != nil {
			return nil, err
		}
		if err := page.SetContent(card); err != nil {
			return nil, err
		}
		png := filepath.Join(dir, fmt.Sprintf("card-%d.png", i))
		if _, err := page.Screenshot(playwright.PageScreenshotOptions{Path: &png}); err != nil {
			return nil, err
		}
		pngs = append(pngs, png)
	}
	return pngs, nil
}

// chapterRow is a step row of chapters.md: step, check, m:ss, caption and
// expectation, a cell holding escaped pipes.
var chapterRow = regexp.MustCompile(`^\| (\d+) \| ((?:[^|\\]|\\.)*) \| (\d+):(\d\d) \| ((?:[^|\\]|\\.)*) \| ((?:[^|\\]|\\.)*) \|$`)

var unescapeCell = strings.NewReplacer(`\|`, "|")

// readChapter reads a step row of chapters.md back, its time to the second.
func readChapter(line string) (Chapter, bool) {
	m := chapterRow.FindStringSubmatch(line)
	if m == nil {
		return Chapter{}, false
	}
	n, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[3])
	sec, _ := strconv.Atoi(m[4])
	return Chapter{
		N:       n,
		Check:   unescapeCell.Replace(m[2]),
		At:      time.Duration(min)*time.Minute + time.Duration(sec)*time.Second,
		Caption: unescapeCell.Replace(m[5]),
		Expect:  unescapeCell.Replace(m[6]),
	}, true
}

// joinChapters renders the joined chapters.md: one section per film, each
// row's time shifted by the cards and films before it. A row's own time is
// read back to the second from its chapters.md, so a joined time may be up
// to a second early; the shift itself is exact, the error never adds up.
func joinChapters(pieces []piece, card time.Duration) (string, error) {
	var sections []section
	var at time.Duration
	for _, p := range pieces {
		at += card
		rows := make([]Chapter, len(p.rows))
		for i, c := range p.rows {
			c.At += at
			rows[i] = c
		}
		sections = append(sections, section{Heading: "##", Title: p.title, Rows: rows})
		at += p.duration
	}
	return renderChapters(sections)
}
