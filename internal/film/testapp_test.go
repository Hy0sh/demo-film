package film_test

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"

	"github.com/Hy0sh/demo-film/internal/scenario"
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

//go:embed testdata/scenarios
var scenarios embed.FS

// loadScenario reads testdata/scenarios/<name>.yaml, fills its {{.Field}}
// placeholders from data (the test servers' URLs, a temp dir...), then
// parses and validates it.
func loadScenario(t *testing.T, name string, data any) *scenario.Scenario {
	t.Helper()
	tpl, err := template.ParseFS(scenarios, "testdata/scenarios/"+name+".yaml")
	if err != nil {
		t.Fatal(err)
	}
	var y bytes.Buffer
	if err := tpl.Execute(&y, data); err != nil {
		t.Fatal(err)
	}
	s, err := scenario.Parse(y.Bytes())
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if errs := scenario.Validate(s); len(errs) > 0 {
		t.Fatalf("%s: test scenario invalid: %v", name, errs)
	}
	return s
}

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
