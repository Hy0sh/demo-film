package film_test

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// appHTML is a tiny single-page app covering the whole vocabulary: a nav
// menu (button parent, link child), form controls, a table with icon-only
// buttons, a hover tooltip, a dialog with a select and a Next button, a
// confirm dialog, and a link opening a new tab.
//
//go:embed testdata/app.html
var appHTML string

//go:embed testdata/docs.html
var docsHTML string

//go:embed testdata/mail.html
var mailHTML string

// newApps serves the app and, on another origin, a mail catcher.
func newApps(t *testing.T) (app, mail *httptest.Server) {
	t.Helper()
	am := http.NewServeMux()
	am.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, docsHTML)
	})
	am.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, appHTML)
	})
	app = httptest.NewServer(am)
	mail = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, mailHTML)
	}))
	t.Cleanup(func() { app.Close(); mail.Close() })
	return app, mail
}
