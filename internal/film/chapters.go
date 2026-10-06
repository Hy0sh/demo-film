package film

import (
	_ "embed"
	"strings"
	"text/template"
	"time"
)

// Chapter is one step of the film, with the moment its caption appears.
type Chapter struct {
	N               int
	Check           string
	At              time.Duration
	Caption, Expect string
}

var cell = strings.NewReplacer("|", `\|`, "\n", " ")

//go:embed chapters.tmpl
var chaptersSource string

var chaptersTemplate = template.Must(template.New("chapters").
	Funcs(template.FuncMap{"cell": cell.Replace, "clock": Clock}).
	Parse(chaptersSource))

// section is one titled table of chapters.md.
type section struct {
	Heading, Title string // Heading: "#" for a film, "##" for a part of a joined one
	Rows           []Chapter
}

func renderChapters(sections []section) (string, error) {
	var b strings.Builder
	err := chaptersTemplate.Execute(&b, sections)
	return b.String(), err
}

// Chapters renders chapters.md: one row per step.
func Chapters(title string, rows []Chapter) (string, error) {
	return renderChapters([]section{{Heading: "#", Title: title, Rows: rows}})
}
