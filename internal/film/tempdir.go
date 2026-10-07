package film

import (
	"os"
	"os/signal"
	"syscall"
)

// tempDir makes a run's working directory in outDir, removed by done. A
// Ctrl-C or SIGTERM kills demo-film before a deferred done runs, so the
// signal removes it too; ttyd (-q) and the browser driver end on their own.
func tempDir(outDir string) (dir string, done func(), err error) {
	dir, err = os.MkdirTemp(outDir, ".demo-film-")
	if err != nil {
		return "", nil, err
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		if s, ok := <-sig; ok {
			os.RemoveAll(dir)
			os.Exit(128 + int(s.(syscall.Signal)))
		}
	}()
	return dir, func() {
		signal.Stop(sig)
		close(sig)
		os.RemoveAll(dir)
	}, nil
}
