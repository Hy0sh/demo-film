package scenario

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// nextButton matches the names of buttons that only move a flow forward.
var nextButton = regexp.MustCompile(`(?i)^(next|suivant|continue|continuer)$`)

// Origin returns scheme://host[:port] of an absolute URL.
func Origin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%q is not an absolute URL", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// ResolveURL resolves a path against base_url; an absolute URL stays as is.
func ResolveURL(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(r).String(), nil
}

// Validate checks a parsed scenario without a browser and returns every
// problem found, empty when the scenario is sound.
func Validate(s *Scenario) []string {
	var errs []string
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }

	if strings.TrimSpace(s.Title) == "" {
		add("title is empty")
	}
	baseOrigin := ""
	if strings.TrimSpace(s.BaseURL) == "" {
		add("base_url is empty")
	} else if o, err := Origin(s.BaseURL); err != nil {
		add("base_url: %v", err)
	} else {
		baseOrigin = o
	}
	if s.Viewport.Width <= 0 || s.Viewport.Height <= 0 {
		add("viewport: width and height must be positive")
	} else if s.Viewport.Width%2 != 0 || s.Viewport.Height%2 != 0 {
		add("viewport: width and height must be even (H.264 requires it)")
	}
	if s.Timeout < 0 {
		add("timeout must be positive")
	}
	if s.Speed < 0.25 || s.Speed > 4 {
		add("speed must be between 0.25 and 4")
	}
	if len(s.Steps) == 0 {
		add("steps is empty")
	}

	current := baseOrigin
	for i, st := range s.Steps {
		step := func(format string, args ...any) { add("step %d: %s", i+1, fmt.Sprintf(format, args...)) }
		if strings.TrimSpace(st.Caption) == "" {
			step("caption is empty")
		}
		if strings.TrimSpace(st.Expect) == "" {
			step("expect is empty")
		}
		if len(st.Do) == 0 {
			step("do has no action")
		}
		if st.Check != "" && len(st.See) == 0 {
			step("check %q needs a non-empty see: an acceptance point must be asserted on screen", st.Check)
		}
		for _, t := range st.See {
			if strings.TrimSpace(t) == "" {
				step("see has an empty text")
			}
		}
		for _, a := range st.Do {
			for _, m := range validateAction(a) {
				step("%s: %s", a, m)
			}
			if a.Kind == Open && baseOrigin != "" {
				abs, err := ResolveURL(s.BaseURL, a.Text)
				if err != nil {
					step("%s: %v", a, err)
					continue
				}
				o, err := Origin(abs)
				if err != nil {
					step("%s: %v", a, err)
					continue
				}
				if i > 0 && o == current {
					step("%s: open on the current origin %s reloads the app; navigate through its UI (open is for step 1 or another origin)", a, current)
				}
				current = o
			}
		}
		if n := len(st.Do); n > 0 {
			last := st.Do[n-1]
			if name := forwardName(last); name != "" {
				step("the last action is a click on %q: a step may not end on a forward button, make it open the next step so the caption matches the screen", name)
			}
		}
	}
	return errs
}

// forwardName returns the name of a click that is a bare "next" button.
func forwardName(a Action) string {
	if a.Kind != Click {
		return ""
	}
	for _, n := range []string{a.Text, a.Name, a.Button} {
		if nextButton.MatchString(strings.TrimSpace(n)) {
			return n
		}
	}
	return ""
}

func validateAction(a Action) []string {
	var e []string
	need := func(ok bool, msg string) {
		if !ok {
			e = append(e, msg)
		}
	}
	if a.Within != "" {
		if a.Within != "dialog" {
			e = append(e, fmt.Sprintf("within: %q is not supported, only dialog", a.Within))
		}
		switch a.Kind {
		case Click, Fill, Select, Hover, Wait:
		default:
			e = append(e, "within is allowed on click, fill, select, hover and wait only")
		}
	}
	if (a.Cut || a.Timeout != 0) && a.Kind != Wait {
		e = append(e, "cut and timeout are allowed on wait only")
	}
	if a.Timeout < 0 {
		e = append(e, "timeout must be positive")
	}
	switch a.Kind {
	case Open, Hover, Wait, Press, Confirm:
		need(strings.TrimSpace(a.Text) != "", "the value is empty")
	case Menu:
		need(len(a.Menu) == 2 && a.Menu[0] != "" && a.Menu[1] != "", "menu needs exactly [Parent label, Child label]")
	case Click:
		shapes := 0
		if a.Text != "" {
			shapes++
		}
		if a.Role != "" || a.Name != "" {
			shapes++
			need(a.Role != "" && a.Name != "", "click by role needs both role and name")
		}
		if a.Row != "" || a.Button != "" || a.Nth != nil {
			shapes++
			need(a.Row != "" && (a.Button != "") != (a.Nth != nil), "click in a row needs row and exactly one of button (name) or button: {nth: N}")
		}
		need(shapes == 1, "click is a text, {role, name} or {row, button}")
	case Fill:
		need(a.Field.Label != "" || a.Field.Rank > 0, "fill needs a field")
	case Select:
		need(a.Field.Label != "" || a.Field.Rank > 0, "select needs a field")
		need(a.Option != "", "select needs an option")
	case Popup:
		need(a.Link != "" && a.URLContains != "", "popup needs click and url_contains")
	}
	return e
}
