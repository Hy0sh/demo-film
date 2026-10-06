// Package preflight checks the external dependencies before a run, so a
// missing one surfaces as one clear message with its install command.
package preflight

import (
	"fmt"
	"os/exec"
	"runtime"
)

// BrowserHint is what to run when the Playwright driver or Chromium is missing.
const BrowserHint = "demo-film install"

// FFmpeg checks that ffmpeg is on PATH.
func FFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found on PATH (needed to assemble the video): %s", ffmpegHint())
	}
	return nil
}

func ffmpegHint() string {
	if runtime.GOOS == "darwin" {
		return "brew install ffmpeg"
	}
	return "sudo apt install ffmpeg (or your distribution's package)"
}

// TTYD checks that ttyd, which serves a terminal demo's shell, is on PATH.
func TTYD() error {
	if _, err := exec.LookPath("ttyd"); err != nil {
		hint := "sudo apt install ttyd (or your distribution's package)"
		if runtime.GOOS == "darwin" {
			hint = "brew install ttyd"
		}
		return fmt.Errorf("ttyd not found on PATH (needed to film a terminal): %s", hint)
	}
	return nil
}

// Browser wraps a Playwright driver/browser failure with the install command.
func Browser(err error) error {
	return fmt.Errorf("playwright is not ready: %w\nrun: %s", err, BrowserHint)
}
