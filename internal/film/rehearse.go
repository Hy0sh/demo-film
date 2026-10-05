package film

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mxschmitt/playwright-go"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// Rehearse plays every step with no pause and no video, asserting each
// step's see texts. On failure it writes rehearse-fail-step<N>.png (full
// page) in outDir and returns the *StepError.
func Rehearse(s *scenario.Scenario, outDir string) error {
	ses, err := launch(s, "")
	if err != nil {
		return err
	}
	defer ses.close()
	r := &runner{s: s, page: ses.page}
	for i, st := range s.Steps {
		err := r.runStep(i+1, st)
		if err == nil {
			continue
		}
		shot := filepath.Join(outDir, fmt.Sprintf("rehearse-fail-step%d.png", i+1))
		if _, serr := ses.page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(shot), FullPage: playwright.Bool(true)}); serr != nil {
			return errors.Join(err, fmt.Errorf("screenshot: %w", serr))
		}
		return fmt.Errorf("%w (screenshot: %s)", err, shot)
	}
	return nil
}
