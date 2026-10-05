package film

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// Film records the scenario: for each step the caption is shown for its
// read time, the actions play, the see texts are asserted, then the caption
// plus "you should see" is held. It writes demo.mp4 and chapters.md in
// outDir. A failing step aborts the run and writes nothing.
func Film(s *scenario.Scenario, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(outDir, ".demo-film-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp) // raw webm and caption PNGs

	ses, err := launch(s, tmp)
	if err != nil {
		return err
	}
	defer ses.close()
	r := &runner{s: s, page: ses.page, filming: true}

	// starts[2k] is when step k's caption appears, starts[2k+1] when its
	// "you should see" does.
	var starts []time.Duration
	var chapters []Chapter
	for i, st := range s.Steps {
		at := time.Since(ses.t0)
		starts = append(starts, at)
		chapters = append(chapters, Chapter{N: i + 1, Check: st.Check, At: at, Caption: st.Caption, Expect: st.Expect})
		r.pause(ReadTime(st.Caption))
		if err := r.runStep(i+1, st); err != nil {
			return err
		}
		starts = append(starts, time.Since(ses.t0))
		r.pause(HoldTime(st.Expect))
	}
	end := time.Since(ses.t0)

	video := ses.page.Video()
	if err := ses.ctx.Close(); err != nil {
		return err
	}
	raw, err := video.Path()
	if err != nil {
		return err
	}

	band, err := ses.browser.NewPage(playwright.BrowserNewPageOptions{
		Viewport: &playwright.Size{Width: s.Viewport.Width, Height: BandHeight},
	})
	if err != nil {
		return err
	}
	var pngs []string
	var windows []Window
	for k, from := range starts {
		st := s.Steps[k/2]
		if err := band.SetContent(bandHTML(s.Labels, k/2+1, len(s.Steps), st, k%2 == 1)); err != nil {
			return err
		}
		png := filepath.Join(tmp, fmt.Sprintf("caption-%d.png", k))
		if _, err := band.Screenshot(playwright.PageScreenshotOptions{Path: &png}); err != nil {
			return err
		}
		to := end + 2*time.Second // past the end of the video
		if k+1 < len(starts) {
			to = starts[k+1]
		}
		pngs = append(pngs, png)
		windows = append(windows, Window{from.Seconds(), to.Seconds()})
	}

	filter := Filter(s.Viewport.Width, s.Viewport.Height, windows)
	if err := assemble(raw, pngs, filter, filepath.Join(outDir, "demo.mp4"), len(windows)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "chapters.md"), []byte(Chapters(s.Title, chapters)), 0o644)
}
