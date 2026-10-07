package film

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A Ctrl-C during a film leaves no working directory next to the video:
// the signal kills demo-film before its deferred removal runs.
func TestTempDirGoesOnInterrupt(t *testing.T) {
	if out := os.Getenv("DEMO_FILM_TEMPDIR_OUT"); out != "" {
		dir, _, err := tempDir(out)
		if err != nil {
			os.Exit(2)
		}
		os.Stdout.WriteString(dir + "\n")
		time.Sleep(time.Minute)
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestTempDirGoesOnInterrupt$")
	cmd.Env = append(os.Environ(), "DEMO_FILM_TEMPDIR_OUT="+t.TempDir())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	dir, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	dir = strings.TrimSpace(dir)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the working directory was not made: %v", err)
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	var exit *exec.ExitError
	if err := cmd.Wait(); !errors.As(err, &exit) || exit.ExitCode() != 128+int(syscall.SIGINT) {
		t.Errorf("exit = %v, want %d", err, 128+int(syscall.SIGINT))
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is still there after the interrupt", dir)
	}
}
