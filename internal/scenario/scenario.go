// Package scenario defines the declarative demo scenario: a YAML document
// in a closed vocabulary, decoded strictly and validated without a browser.
package scenario

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultWidth   = 1440
	defaultHeight  = 900
	defaultTimeout = 15 // seconds, per action
)

// Scenario is one demo, from the first page to the last "you should see".
type Scenario struct {
	Title    string   `yaml:"title"`
	BaseURL  string   `yaml:"base_url"`
	Viewport Viewport `yaml:"viewport"`
	Locale   string   `yaml:"locale"`
	Hide     []string `yaml:"hide"`
	// Timeout is the budget of one action, in seconds.
	Timeout int `yaml:"timeout"`
	// Speed scales the gestures (cursor travel, pauses around actions,
	// typing): 1 is the default, 0.5 twice as slow, 2 twice as fast.
	// Caption read times are not affected.
	Speed  float64 `yaml:"speed"`
	Labels Labels  `yaml:"labels"`
	Steps  []Step  `yaml:"steps"`
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
