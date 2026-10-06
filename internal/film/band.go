package film

import (
	_ "embed"
	"html/template"
	"strings"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// BandHeight is the height of the caption band under the page, in pixels.
const BandHeight = 150

const (
	bandBackground = "#11161b"
	bandAccent     = "#2457a6"
	bandHeader     = "#7fa8e8"
)

//go:embed band.html
var bandSource string

var bandTemplate = template.Must(template.New("band").Parse(bandSource))

// bandHTML is one state of the band: the caption alone (state A), or the
// caption plus "you should see" (state B).
func bandHTML(l scenario.Labels, n, total int, st scenario.Step, withExpect bool) (string, error) {
	data := struct {
		Step, CheckLabel, See, Check, Caption, Expect string
		N, Total, Height                              int
		Background, Accent, Header                    string
	}{
		Step: l.Step, CheckLabel: l.Check, See: l.See, Check: st.Check, Caption: st.Caption,
		N: n, Total: total, Height: BandHeight,
		Background: bandBackground, Accent: bandAccent, Header: bandHeader,
	}
	if withExpect {
		data.Expect = st.Expect
	}
	var b strings.Builder
	err := bandTemplate.Execute(&b, data)
	return b.String(), err
}
