package film

import (
	"fmt"
	"strings"
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

// Chapters renders chapters.md: one row per step.
func Chapters(title string, rows []Chapter) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n| Step | Check | Time | Caption | Expect |\n|---|---|---|---|---|\n", title)
	for _, c := range rows {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", c.N, cell.Replace(c.Check), Clock(c.At), cell.Replace(c.Caption), cell.Replace(c.Expect))
	}
	return b.String()
}
