package film

import (
	"fmt"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/Hy0sh/demo-film/internal/preflight"
	"github.com/Hy0sh/demo-film/internal/scenario"
)

// session is one headless Chromium with a single page.
type session struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	ctx     playwright.BrowserContext
	page    playwright.Page
	t0      time.Time // when the page, hence the video, started
}

// startBrowser starts the Playwright driver and a headless Chromium with
// those command-line arguments.
func startBrowser(args ...string) (*playwright.Playwright, playwright.Browser, error) {
	pw, err := playwright.Run(&playwright.RunOptions{Verbose: false})
	if err != nil {
		return nil, nil, preflight.Browser(err)
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Args: args})
	if err != nil {
		pw.Stop()
		return nil, nil, preflight.Browser(err)
	}
	return pw, browser, nil
}

// launch starts Chromium. A non-empty videoDir records the page there;
// creds, when set, answer the HTTP authentication of the page (a terminal).
func launch(s *scenario.Scenario, videoDir string, creds *playwright.HttpCredentials) (*session, error) {
	var args []string
	if s.Locale != "" {
		// Native inputs (dates, months) are formatted in the browser
		// process's language, which the context's locale does not reach.
		args = append(args, "--lang="+s.Locale)
	}
	pw, browser, err := startBrowser(args...)
	if err != nil {
		return nil, err
	}
	vp := &playwright.Size{Width: s.Viewport.Width, Height: s.Viewport.Height}
	opts := playwright.BrowserNewContextOptions{Viewport: vp, HttpCredentials: creds}
	if s.Locale != "" {
		// The browser's own locale formats native inputs (dates, months) and
		// sets Accept-Language, as i18next alone cannot.
		opts.Locale = playwright.String(s.Locale)
	}
	if videoDir != "" {
		opts.RecordVideo = &playwright.RecordVideo{Dir: playwright.String(videoDir), Size: vp}
	}
	ses := &session{pw: pw, browser: browser}
	if ses.ctx, err = browser.NewContext(opts); err != nil {
		ses.close()
		return nil, fmt.Errorf("new context: %w", err)
	}
	script := InitScript(s)
	if err := ses.ctx.AddInitScript(playwright.Script{Content: &script}); err != nil {
		ses.close()
		return nil, err
	}
	if ses.page, err = ses.ctx.NewPage(); err != nil {
		ses.close()
		return nil, fmt.Errorf("new page: %w", err)
	}
	ses.t0 = time.Now()
	ses.page.SetDefaultTimeout(float64(s.Timeout) * 1000)
	return ses, nil
}

func (s *session) close() {
	if s.browser != nil {
		s.browser.Close()
	}
	s.pw.Stop()
}
