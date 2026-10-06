// demo-film films a demo of a web app headless from a declarative YAML
// scenario, and outputs an mp4 with a caption band under the page plus a
// chapter list.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mxschmitt/playwright-go"
	"github.com/spf13/cobra"

	"github.com/Hy0sh/demo-film/internal/film"
	"github.com/Hy0sh/demo-film/internal/preflight"
	"github.com/Hy0sh/demo-film/internal/scenario"
	"github.com/Hy0sh/demo-film/internal/version"
)

func main() {
	if err := newRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "demo-film",
		Short:         "Film a web app demo from a declarative scenario",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("demo-film {{.Version}}\n")
	root.SetErr(os.Stderr)
	root.AddCommand(checkCmd(), rehearseCmd(), filmCmd(), joinCmd(), installCmd())
	return root
}

// fail prints the error the way every command reports it, and returns it so
// cobra exits non-zero.
func fail(err error) error {
	fmt.Fprintln(os.Stderr, "demo-film:", err)
	return err
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <scenario.yaml>",
		Short: "Validate a scenario without a browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := scenario.Load(args[0])
			if err != nil {
				return fail(err)
			}
			fmt.Printf("%s: ok, %d steps\n", args[0], len(s.Steps))
			return nil
		},
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the Playwright driver and Chromium",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}, Verbose: true})
			if err != nil {
				return fail(errors.Join(errors.New("install failed"), err))
			}
			return nil
		},
	}
}

func rehearseCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "rehearse <scenario.yaml>",
		Short: "Run every step with no pause and no video, asserting each see",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := scenario.Load(args[0])
			if err != nil {
				return fail(err)
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fail(err)
			}
			if err := film.Rehearse(s, outDir); err != nil {
				return fail(err)
			}
			fmt.Printf("rehearsal ok: %d steps\n", len(s.Steps))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "directory for the failure screenshot")
	return cmd
}

func filmCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "film <scenario.yaml>",
		Short: "Record the demo: demo.mp4 with a caption band, and chapters.md",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := scenario.Load(args[0])
			if err != nil {
				return fail(err)
			}
			if err := preflight.FFmpeg(); err != nil {
				return fail(err)
			}
			if err := film.Film(s, outDir); err != nil {
				return fail(err)
			}
			fmt.Println(filepath.Join(outDir, "demo.mp4"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "output directory")
	return cmd
}

func joinCmd() *cobra.Command {
	var outDir string
	var noCards bool
	cmd := &cobra.Command{
		Use:   "join <film-dir> <film-dir>... -o <dir>",
		Short: "Join the outputs of several films into one, a title card before each",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := preflight.FFmpeg(); err != nil {
				return fail(err)
			}
			if err := film.Join(args, outDir, !noCards); err != nil {
				return fail(err)
			}
			fmt.Println(filepath.Join(outDir, "demo.mp4"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "output directory")
	cmd.Flags().BoolVar(&noCards, "no-cards", false, "no title card before each film")
	return cmd
}
