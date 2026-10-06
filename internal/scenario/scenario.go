// Package scenario defines the declarative demo scenario: a YAML document
// in a closed vocabulary, decoded strictly and validated without a browser.
package scenario

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultWidth   = 1440
	defaultHeight  = 900
	defaultTimeout = 15 // seconds, per action
)

// Scenario is one demo, from the first page to the last "you should see".
type Scenario struct {
	Title   string `yaml:"title"`
	BaseURL string `yaml:"base_url"`
	// Terminal films a shell instead of a web app: it replaces base_url.
	Terminal *Terminal `yaml:"terminal"`
	Viewport Viewport  `yaml:"viewport"`
	Locale   string    `yaml:"locale"`
	Hide     []string  `yaml:"hide"`
	// Timeout is the budget of one action, in seconds.
	Timeout int `yaml:"timeout"`
	// Speed scales the whole video, gestures (cursor travel, pauses around
	// actions, typing) and caption read/hold times: 1 is the default, 0.5
	// twice as slow, 2 twice as fast.
	Speed     float64    `yaml:"speed"`
	Labels    Labels     `yaml:"labels"`
	Watermark *Watermark `yaml:"watermark"`
	Steps     []Step     `yaml:"steps"`
}

// Watermark signs the whole video: an image or a line of text, in a corner
// of the page area, never over the caption band.
type Watermark struct {
	Image    string  `yaml:"image"` // relative to the scenario file
	Text     string  `yaml:"text"`
	Position string  `yaml:"position"` // top-left, top-right, bottom-left, bottom-right
	Opacity  float64 `yaml:"opacity"`  // 0 to 1
}

// Terminal is the shell a terminal demo films, served in the browser by ttyd.
type Terminal struct {
	Shell string `yaml:"shell"` // command line; default: $SHELL
	Cwd   string `yaml:"cwd"`   // relative to the scenario file, ~ expanded; default: the current directory
}

type Viewport struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// Labels are the words of the caption band, English unless overridden.
type Labels struct {
	Step  string `yaml:"step"`
	See   string `yaml:"see"`
	Check string `yaml:"check"`
	Later string `yaml:"later"` // the card after a cut: "⏩ 2:14 later"
}

// Step is one caption, the actions it films, and what must be on screen after.
type Step struct {
	Caption string   `yaml:"caption"`
	Check   string   `yaml:"check"`
	Do      []Action `yaml:"do"`
	See     []string `yaml:"see"`
	Expect  string   `yaml:"expect"`
}

// Parse decodes a scenario strictly (an unknown key is an error) and
// applies the defaults. It does not validate: see Validate.
func Parse(data []byte) (*Scenario, error) {
	var s Scenario
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	if s.Viewport.Width == 0 && s.Viewport.Height == 0 {
		s.Viewport = Viewport{Width: defaultWidth, Height: defaultHeight}
	}
	if s.Timeout == 0 {
		s.Timeout = defaultTimeout
	}
	if s.Speed == 0 {
		s.Speed = 1
	}
	if s.Labels.Step == "" {
		s.Labels.Step = "Step"
	}
	if s.Labels.See == "" {
		s.Labels.See = "You should see:"
	}
	if s.Labels.Check == "" {
		s.Labels.Check = "check"
	}
	if s.Labels.Later == "" {
		s.Labels.Later = "later"
	}
	if w := s.Watermark; w != nil {
		if w.Position == "" {
			w.Position = "bottom-right"
		}
		if w.Opacity == 0 {
			w.Opacity = 0.6
		}
	}
	return &s, nil
}

// Load reads, parses and validates a scenario file. The error lists every
// problem validation found.
func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	// Paths in a scenario are relative to its file, not to where it is run.
	if w := s.Watermark; w != nil && w.Image != "" && !filepath.IsAbs(w.Image) {
		w.Image = filepath.Join(filepath.Dir(path), w.Image)
	}
	if t := s.Terminal; t != nil && t.Cwd != "" && !filepath.IsAbs(t.Cwd) && !strings.HasPrefix(t.Cwd, "~") {
		t.Cwd = filepath.Join(filepath.Dir(path), t.Cwd)
	}
	if errs := Validate(s); len(errs) > 0 {
		return nil, ValidationError{Path: path, Errs: errs}
	}
	return s, nil
}

// ValidationError carries every problem of a scenario.
type ValidationError struct {
	Path string
	Errs []string
}

func (e ValidationError) Error() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s: %d problem(s)", e.Path, len(e.Errs))
	for _, m := range e.Errs {
		b.WriteString("\n  - " + m)
	}
	return b.String()
}
