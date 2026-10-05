package preflight

import (
	"strings"
	"testing"
)

func TestFFmpegMissingNamesTheInstall(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := FFmpeg()
	if err == nil || !strings.Contains(err.Error(), "ffmpeg") || !strings.Contains(err.Error(), "install") {
		t.Fatalf("want an error with an install hint, got %v", err)
	}
}

func TestBrowserHintsTheInstallCommand(t *testing.T) {
	err := Browser(errString("boom"))
	if !strings.Contains(err.Error(), "demo-film install") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v", err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
