package film

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Hy0sh/demo-film/internal/preflight"
	"github.com/Hy0sh/demo-film/internal/scenario"
)

//go:embed prompt.js
var promptJS string

// ttydOptions set up xterm.js for filming: the DOM renderer puts the text in
// the page, where wait and see find it (the default canvas hides it), and no
// size overlay or leave prompt pollutes the video.
var ttydOptions = []string{
	"rendererType=dom",
	"fontSize=18",
	"disableResizeOverlay=true",
	"disableLeaveAlert=true",
}

// withTerminal serves the scenario's shell with ttyd, on the loopback only
// (the shell is writable), and returns a copy of the scenario pointed at it
// whose step 1 opens it first, so it loads off camera. A web scenario is
// returned as is. stop ends ttyd, hence the shell.
func withTerminal(s *scenario.Scenario) (_ *scenario.Scenario, stop func(), err error) {
	if s.Terminal == nil {
		return s, func() {}, nil
	}
	if err := preflight.TTYD(); err != nil {
		return nil, nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, nil, err
	}
	// -q: ttyd exits once the browser is gone, even when demo-film is killed
	// before stop runs, so no writable shell outlives the run.
	args := []string{"-i", "127.0.0.1", "-p", strconv.Itoa(port), "-W", "-O", "-q"}
	for _, o := range ttydOptions {
		args = append(args, "-t", o)
	}
	if s.Terminal.Cwd != "" {
		args = append(args, "-w", expandHome(s.Terminal.Cwd))
	}
	shell := s.Terminal.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	args = append(args, strings.Fields(shell)...)
	cmd := exec.Command("ttyd", args...)
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("ttyd: %w", err)
	}
	stop = func() { cmd.Process.Kill(); cmd.Wait() }

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitUp(url, 10*time.Second); err != nil {
		stop()
		return nil, nil, fmt.Errorf("ttyd: %w", err)
	}
	t := *s
	t.BaseURL = url
	first := t.Steps[0]
	first.Do = append([]scenario.Action{{Kind: scenario.Open, Text: "/"}}, first.Do...)
	t.Steps = append([]scenario.Step{first}, t.Steps[1:]...)
	return &t, stop, nil
}

// freePort asks the system for a free loopback port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitUp(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("not answering on %s after %s: %w", url, timeout, err)
		}
		time.Sleep(pollEvery)
	}
}

func expandHome(p string) string {
	if home, err := os.UserHomeDir(); err == nil && (p == "~" || strings.HasPrefix(p, "~/")) {
		return filepath.Join(home, p[1:])
	}
	return p
}
