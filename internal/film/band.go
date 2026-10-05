package film

import (
	"fmt"
	"html"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// BandHeight is the height of the caption band under the page, in pixels.
const BandHeight = 150

const (
	bandBackground = "#11161b"
	bandAccent     = "#2457a6"
	bandHeader     = "#7fa8e8"
)

// bandHTML is one state of the band: the caption alone (state A), or the
// caption plus "you should see" (state B).
func bandHTML(l scenario.Labels, n, total int, st scenario.Step, withExpect bool) string {
	head := fmt.Sprintf("%s %d/%d", html.EscapeString(l.Step), n, total)
	if st.Check != "" {
		head += fmt.Sprintf(" · %s %s", html.EscapeString(l.Check), html.EscapeString(st.Check))
	}
	body := fmt.Sprintf(`<div style="font-weight:700;color:%s">%s</div><div>%s</div>`, bandHeader, head, html.EscapeString(st.Caption))
	if withExpect {
		body += fmt.Sprintf(`<div style="margin-top:2px"><b>%s</b> %s</div>`, html.EscapeString(l.See), html.EscapeString(st.Expect))
	}
	return fmt.Sprintf(`<!doctype html><meta charset="utf-8"><body style="margin:0;background:%s;color:#fff;font:19px/1.4 system-ui,sans-serif;height:%dpx;box-sizing:border-box;padding:12px 28px;border-top:4px solid %s;overflow:hidden">%s</body>`,
		bandBackground, BandHeight, bandAccent, body)
}
