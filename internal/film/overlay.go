package film

import (
	_ "embed"
	"encoding/json"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

//go:embed overlay.js
var overlayJS string

// overlayConfig is what overlay.js receives.
type overlayConfig struct {
	Locale string   `json:"locale"`
	Hide   []string `json:"hide"`
}

// InitScript is injected before any page script: overlay.js called with the
// scenario's locale and hidden elements.
func InitScript(s *scenario.Scenario) string {
	cfg := overlayConfig{Locale: s.Locale, Hide: s.Hide}
	if cfg.Hide == nil {
		cfg.Hide = []string{}
	}
	arg, _ := json.Marshal(cfg)
	return overlayJS + "(" + string(arg) + ");\n"
}
