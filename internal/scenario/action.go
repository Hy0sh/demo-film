package scenario

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Action kinds: the whole vocabulary.
const (
	Open    = "open"
	Menu    = "menu"
	Click   = "click"
	Fill    = "fill"
	Select  = "select"
	Press   = "press"
	Type    = "type" // keystrokes to the focused element, a terminal above all
	Hover   = "hover"
	Wait    = "wait"
	Popup   = "popup"
	Confirm = "confirm"
)

// What a wait waits for, besides a text showing up.
const (
	Gone    = "gone"    // no visible element shows the text any more
	Enabled = "enabled" // the control showing the text is no longer disabled
)

// Action is one item of a step's `do`: a single-key map, plus the optional
// `within`, `cut` and `timeout` modifiers. Only the fields of its Kind are set.
type Action struct {
	Kind    string
	Within  string // "" or "dialog"
	Cut     bool   // wait: the video skips from the start of the wait to the text
	Timeout int    // wait: seconds, overrides the scenario's timeout
	Until   string // wait: "" for the text to show up, Gone or Enabled

	Text string   // open (url or path), click/hover/wait text, press key, confirm button
	Menu []string // menu: parent, child

	Role, Name string // click by role
	Match      *int   // click by text or role: which visible match, from 0 (negative counts from the end)
	Row        string // click in a table row...
	Button     string // ...the button, by accessible name
	Nth        *int   // ...or by index (negative counts from the end)

	Field  Field  // fill, select
	Value  string // fill
	Option string // select

	Link, URLContains string // popup
}

// Field designates a form control: by label or placeholder, or by rank
// (1-based among the visible controls of its kind).
type Field struct {
	Label string
	Rank  int
}

func (f Field) String() string {
	if f.Rank > 0 {
		return fmt.Sprintf("#%d", f.Rank)
	}
	return fmt.Sprintf("%q", f.Label)
}

func (f *Field) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: field is a label, a placeholder or a rank", n.Line)
	}
	if n.Tag == "!!int" {
		if err := n.Decode(&f.Rank); err != nil {
			return err
		}
		if f.Rank < 1 {
			return fmt.Errorf("line %d: field rank starts at 1", n.Line)
		}
		return nil
	}
	return n.Decode(&f.Label)
}

func decodeStrict(n *yaml.Node, v any) error {
	b, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}
	return nil
}

func (a *Action) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: an action is a map with one key (open, menu, click...)", n.Line)
	}
	var keys []string
	var key, val *yaml.Node
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i].Value
		var modifier any
		switch k {
		case "within":
			modifier = &a.Within
		case "cut":
			modifier = &a.Cut
		case "timeout":
			modifier = &a.Timeout
		}
		if modifier != nil {
			if err := n.Content[i+1].Decode(modifier); err != nil {
				return err
			}
			continue
		}
		keys = append(keys, k)
		key, val = n.Content[i], n.Content[i+1]
	}
	if len(keys) != 1 {
		sort.Strings(keys)
		return fmt.Errorf("line %d: an action has exactly one verb, got %d (%s)", n.Line, len(keys), strings.Join(keys, ", "))
	}
	a.Kind = key.Value
	switch a.Kind {
	case Wait:
		return a.decodeWait(val)
	case Open, Hover, Press, Type, Confirm:
		return val.Decode(&a.Text)
	case Menu:
		if err := val.Decode(&a.Menu); err != nil {
			return fmt.Errorf("line %d: menu is a list [Parent label, Child label]", val.Line)
		}
	case Click:
		return a.decodeClick(val)
	case Fill:
		var f struct {
			Field Field  `yaml:"field"`
			Value string `yaml:"value"`
		}
		if err := decodeStrict(val, &f); err != nil {
			return err
		}
		a.Field, a.Value = f.Field, f.Value
	case Select:
		var f struct {
			Field  Field  `yaml:"field"`
			Option string `yaml:"option"`
		}
		if err := decodeStrict(val, &f); err != nil {
			return err
		}
		a.Field, a.Option = f.Field, f.Option
	case Popup:
		var f struct {
			Click       string `yaml:"click"`
			URLContains string `yaml:"url_contains"`
		}
		if err := decodeStrict(val, &f); err != nil {
			return err
		}
		a.Link, a.URLContains = f.Click, f.URLContains
	default:
		return fmt.Errorf("line %d: unknown action %q (open, menu, click, fill, select, press, type, hover, wait, popup, confirm)", key.Line, a.Kind)
	}
	return nil
}

// decodeWait reads the shapes of wait: a text to show up, {gone: text} for
// a text to disappear, {enabled: text} for a control to become usable.
func (a *Action) decodeWait(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		return n.Decode(&a.Text)
	}
	var w struct {
		Gone    string `yaml:"gone"`
		Enabled string `yaml:"enabled"`
	}
	if err := decodeStrict(n, &w); err != nil {
		return err
	}
	switch {
	case w.Gone != "" && w.Enabled == "":
		a.Until, a.Text = Gone, w.Gone
	case w.Enabled != "" && w.Gone == "":
		a.Until, a.Text = Enabled, w.Enabled
	default:
		return fmt.Errorf("line %d: wait is a text, {gone: text} or {enabled: text}", n.Line)
	}
	return nil
}

// decodeClick reads the shapes of click: a text, {text, nth},
// {role, name[, nth]}, or {row, button} where button is a name or {nth: N}.
func (a *Action) decodeClick(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		return n.Decode(&a.Text)
	}
	var c struct {
		Text   string    `yaml:"text"`
		Role   string    `yaml:"role"`
		Name   string    `yaml:"name"`
		Nth    *int      `yaml:"nth"`
		Row    string    `yaml:"row"`
		Button yaml.Node `yaml:"button"`
	}
	if err := decodeStrict(n, &c); err != nil {
		return err
	}
	a.Text, a.Role, a.Name, a.Match, a.Row = c.Text, c.Role, c.Name, c.Nth, c.Row
	switch c.Button.Kind {
	case 0:
	case yaml.ScalarNode:
		a.Button = c.Button.Value
	case yaml.MappingNode:
		var b struct {
			Nth *int `yaml:"nth"`
		}
		if err := decodeStrict(&c.Button, &b); err != nil {
			return err
		}
		if b.Nth == nil {
			return fmt.Errorf("line %d: button is a name or {nth: N}", c.Button.Line)
		}
		a.Nth = b.Nth
	default:
		return fmt.Errorf("line %d: button is a name or {nth: N}", c.Button.Line)
	}
	return nil
}

// String names the action in failure messages.
func (a Action) String() string {
	var s string
	switch a.Kind {
	case Menu:
		s = fmt.Sprintf("menu %q", a.Menu)
	case Click:
		switch {
		case a.Row != "" && a.Nth != nil:
			s = fmt.Sprintf("click button #%d of the row %q", *a.Nth, a.Row)
		case a.Row != "":
			s = fmt.Sprintf("click button %q of the row %q", a.Button, a.Row)
		case a.Role != "":
			s = fmt.Sprintf("click %s %q", a.Role, a.Name)
		default:
			s = fmt.Sprintf("click %q", a.Text)
		}
		if a.Match != nil {
			s += fmt.Sprintf(" (match #%d)", *a.Match)
		}
	case Fill:
		s = fmt.Sprintf("fill %s with %q", a.Field, a.Value)
	case Select:
		s = fmt.Sprintf("select %q in %s", a.Option, a.Field)
	case Popup:
		s = fmt.Sprintf("popup via %q expecting a URL containing %q", a.Link, a.URLContains)
	case Wait:
		s = fmt.Sprintf("wait %q", a.Text)
		if a.Until != "" {
			s = fmt.Sprintf("wait until %q is %s", a.Text, a.Until)
		}
	default:
		s = fmt.Sprintf("%s %q", a.Kind, a.Text)
	}
	if a.Within != "" {
		s += " within " + a.Within
	}
	if a.Cut {
		s += " (cut)"
	}
	return s
}
