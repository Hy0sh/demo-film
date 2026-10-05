// demo-film films a demo of a web app headless from a declarative YAML
// scenario, and outputs an mp4 with a caption band under the page plus a
// chapter list.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"

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
	root.AddCommand(checkCmd(), installCmd())
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
