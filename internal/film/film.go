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

	// Pre-roll: when step 1 opens with `open`, the app loads off camera and
	// the video is cut from the moment it shows, not from the blank page.
	steps := s.Steps
	var offset time.Duration
	if first := steps[0]; len(first.Do) > 0 && first.Do[0].Kind == scenario.Open {
		if err := r.do(first.Do[0]); err != nil {
			return &StepError{Step: 1, Caption: first.Caption, Action: first.Do[0].String(), Err: err}
		}
		if err := ses.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle}); err != nil {
			return &StepError{Step: 1, Caption: first.Caption, Action: first.Do[0].String(), Err: err}
		}
		offset = time.Since(ses.t0)
		first.Do = first.Do[1:]
		steps = append([]scenario.Step{first}, steps[1:]...)
	}
	since := func() time.Duration { return time.Since(ses.t0) - offset }

	// starts[2k] is when step k's caption appears, starts[2k+1] when its
	// "you should see" does.
	var starts []time.Duration
	var chapters []Chapter
	for i, st := range steps {
		at := since()
		starts = append(starts, at)
		chapters = append(chapters, Chapter{N: i + 1, Check: st.Check, At: at, Caption: st.Caption, Expect: st.Expect})
		r.pause(ReadTime(st.Caption))
		if err := r.runStep(i+1, st); err != nil {
			return err
		}
		starts = append(starts, since())
		r.pause(HoldTime(st.Expect))
	}
	end := since()

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
	if err := assemble(raw, offset, pngs, filter, filepath.Join(outDir, "demo.mp4"), len(windows)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "chapters.md"), []byte(Chapters(s.Title, chapters)), 0o644)
}
