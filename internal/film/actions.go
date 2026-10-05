package film

import (
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// Pacing when filming, tuned for a human eye following the cursor.
const (
	pollEvery   = 100 * time.Millisecond
	typingDelay = 70.0 // ms per character
	travelTime  = 700 * time.Millisecond
	travelFrame = 20 * time.Millisecond
	arrivalWait = 600 * time.Millisecond // the eye lands on the target before the click
	afterAction = 700 * time.Millisecond // the effect is seen before the next gesture
)

// runner plays the actions of a scenario on a page. Filming adds the
// pauses and the visible typing a viewer needs; rehearsing skips them.
type runner struct {
	s       *scenario.Scenario
	page    playwright.Page
	filming bool
	x, y    float64 // last cursor position, where the next travel starts
}

func (r *runner) timeout() time.Duration { return time.Duration(r.s.Timeout) * time.Second }

func (r *runner) pause(d time.Duration) {
	if r.filming {
		time.Sleep(d) // the recording runs meanwhile: the pause is the point
	}
}

// gesture scales a gesture duration by the scenario's speed.
func (r *runner) gesture(d time.Duration) time.Duration {
	return time.Duration(float64(d) / r.s.Speed)
}

// ActionPace is the filming time one pointer action adds at that speed:
// cursor travel, arrival wait and the pause after it.
func ActionPace(speed float64) time.Duration {
	return time.Duration(float64(travelTime+arrivalWait+afterAction) / speed)
}

// root is where lookups start: the page, or the last open dialog.
func (r *runner) root(a scenario.Action) playwright.Locator {
	if a.Within == "dialog" {
		return r.page.Locator("[role=dialog]").Last()
	}
	return r.page.Locator("body")
}

// firstVisible returns the first visible element of the first candidate
// that has one, polling until the timeout. Candidates are ordered from the
// strictest (exact text) to the loosest (case-insensitive substring).
func (r *runner) firstVisible(timeout time.Duration, cands ...playwright.Locator) (playwright.Locator, bool) {
	deadline := time.Now().Add(timeout)
	for {
		for _, c := range cands {
			vis := c.Locator("visible=true")
			if n, err := vis.Count(); err == nil && n > 0 {
				return vis.First(), true
			}
		}
		if !time.Now().Before(deadline) {
			return nil, false
		}
		time.Sleep(pollEvery)
	}
}

func (r *runner) find(timeout time.Duration, what string, cands ...playwright.Locator) (playwright.Locator, error) {
	loc, ok := r.firstVisible(timeout, cands...)
	if !ok {
		return nil, fmt.Errorf("nothing visible matched %s within %s", what, timeout)
	}
	return loc, nil
}

func (r *runner) byText(root playwright.Locator, text string) []playwright.Locator {
	return []playwright.Locator{
		root.GetByText(text, playwright.LocatorGetByTextOptions{Exact: playwright.Bool(true)}),
		root.GetByText(text, playwright.LocatorGetByTextOptions{Exact: playwright.Bool(false)}),
	}
}

func (r *runner) byName(root playwright.Locator, role, name string) []playwright.Locator {
	var out []playwright.Locator
	for _, exact := range []bool{true, false} {
		out = append(out, root.GetByRole(playwright.AriaRole(role), playwright.LocatorGetByRoleOptions{Name: name, Exact: playwright.Bool(exact)}))
	}
	return out
}

// travel moves the visible cursor to the centre of the element.
func (r *runner) travel(loc playwright.Locator) error {
	if err := loc.ScrollIntoViewIfNeeded(); err != nil {
		return err
	}
	box, err := loc.BoundingBox()
	if err != nil || box == nil {
		return fmt.Errorf("element has no box: %v", err)
	}
	tx, ty := box.X+box.Width/2, box.Y+box.Height/2
	if r.filming {
		// Playwright's Steps option sends every step at once: animate in real time.
		n := max(int(r.gesture(travelTime)/travelFrame), 1)
		for i := 1; i <= n; i++ {
			k := easeInOut(float64(i) / float64(n))
			if err := r.page.Mouse().Move(r.x+(tx-r.x)*k, r.y+(ty-r.y)*k); err != nil {
				return err
			}
			time.Sleep(travelFrame)
		}
	} else if err := r.page.Mouse().Move(tx, ty); err != nil {
		return err
	}
	r.x, r.y = tx, ty
	r.pause(r.gesture(arrivalWait))
	return nil
}

// easeInOut maps linear progress in [0,1] to a smooth start and stop.
func easeInOut(t float64) float64 {
	if t < .5 {
		return 2 * t * t
	}
	return 1 - (-2*t+2)*(-2*t+2)/2
}

func (r *runner) clickOn(loc playwright.Locator) error {
	if err := r.travel(loc); err != nil {
		return err
	}
	return loc.Click()
}

// controls are the form controls a numeric rank counts among.
const controls = "input:not([type=hidden]):not([type=checkbox]):not([type=radio]):not([type=submit]):not([type=button]),textarea"

func (r *runner) field(root playwright.Locator, f scenario.Field, selects bool) []playwright.Locator {
	base := controls
	if selects {
		base = "select"
	}
	switch {
	case f.Rank > 0:
		return []playwright.Locator{root.Locator(base).Locator("visible=true").Nth(f.Rank - 1)}
	case f.Label == "password" && !selects:
		return []playwright.Locator{root.Locator("input[type=password]")}
	}
	var out []playwright.Locator
	for _, exact := range []bool{true, false} {
		byLabel := root.GetByLabel(f.Label, playwright.LocatorGetByLabelOptions{Exact: playwright.Bool(exact)})
		if selects {
			out = append(out, byLabel)
			continue
		}
		out = append(out, byLabel.Or(root.GetByPlaceholder(f.Label, playwright.LocatorGetByPlaceholderOptions{Exact: playwright.Bool(exact)})))
	}
	return out
}

// do plays one action.
func (r *runner) do(a scenario.Action) error {
	root := r.root(a)
	to := r.timeout()
	switch a.Kind {
	case scenario.Open:
		url, err := scenario.ResolveURL(r.s.BaseURL, a.Text)
		if err != nil {
			return err
		}
		_, err = r.page.Goto(url)
		return err

	case scenario.Menu:
		return r.menu(root, a.Menu[0], a.Menu[1])

	case scenario.Click:
		loc, err := r.clickTarget(root, a)
		if err != nil {
			return err
		}
		return r.clickOn(loc)

	case scenario.Fill:
		loc, err := r.find(to, "field "+a.Field.String(), r.field(root, a.Field, false)...)
		if err != nil {
			return err
		}
		if err := r.clickOn(loc); err != nil {
			return err
		}
		if !r.filming {
			return loc.Fill(a.Value)
		}
		if err := loc.Fill(""); err != nil {
			return err
		}
		return loc.PressSequentially(a.Value, playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(typingDelay / r.s.Speed)})

	case scenario.Select:
		return r.selectOption(root, a)

	case scenario.Press:
		return r.page.Keyboard().Press(a.Text)

	case scenario.Hover:
		loc, err := r.find(to, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
		if err != nil {
			return err
		}
		return r.travel(loc)

	case scenario.Wait:
		_, err := r.find(to, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
		return err

	case scenario.Popup:
		return r.popup(root, a)

	case scenario.Confirm:
		dlg := r.page.Locator("[role=dialog]").Last()
		loc, err := r.find(to, fmt.Sprintf("button %q in the last dialog", a.Text), r.byName(dlg, "button", a.Text)...)
		if err != nil {
			return err
		}
		return r.clickOn(loc)
	}
	return fmt.Errorf("unknown action %q", a.Kind)
}

func (r *runner) clickTarget(root playwright.Locator, a scenario.Action) (playwright.Locator, error) {
	to := r.timeout()
	switch {
	case a.Row != "":
		row := root.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: a.Row})
		if a.Nth != nil {
			return r.find(to, fmt.Sprintf("button #%d in the row %q", *a.Nth, a.Row),
				row.First().Locator("button").Locator("visible=true").Nth(*a.Nth))
		}
		return r.find(to, fmt.Sprintf("button %q in the row %q", a.Button, a.Row),
			r.byName(row.First(), "button", a.Button)...)
	case a.Role != "":
		return r.find(to, fmt.Sprintf("%s %q", a.Role, a.Name), r.byName(root, a.Role, a.Name)...)
	}
	return r.find(to, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
}

// entry is a navigation entry: a link or a button, by exact accessible name.
// No substring fallback: a hidden "Settings" child would otherwise match a
// visible "General settings" entry elsewhere in the menu.
func (r *runner) entry(root playwright.Locator, name string) []playwright.Locator {
	opts := playwright.LocatorGetByRoleOptions{Name: name, Exact: playwright.Bool(true)}
	return []playwright.Locator{root.GetByRole("link", opts).Or(root.GetByRole("button", opts))}
}

// menu clicks the child entry, opening its parent first when it is hidden.
func (r *runner) menu(root playwright.Locator, parent, child string) error {
	if _, ok := r.firstVisible(0, r.entry(root, child)...); !ok {
		p, err := r.find(r.timeout(), fmt.Sprintf("menu entry %q", parent), r.entry(root, parent)...)
		if err != nil {
			return err
		}
		if err := r.clickOn(p); err != nil {
			return err
		}
	}
	c, err := r.find(r.timeout(), fmt.Sprintf("menu entry %q", child), r.entry(root, child)...)
	if err != nil {
		return err
	}
	return r.clickOn(c)
}

func (r *runner) selectOption(root playwright.Locator, a scenario.Action) error {
	loc, err := r.find(r.timeout(), "select "+a.Field.String(), r.field(root, a.Field, true)...)
	if err != nil {
		return err
	}
	if err := r.travel(loc); err != nil {
		return err
	}
	opts, err := loc.Locator("option").AllInnerTexts()
	if err != nil {
		return err
	}
	label, ok := pickOption(opts, a.Option)
	if !ok {
		return fmt.Errorf("select %s has no option %q (options: %q)", a.Field, a.Option, opts)
	}
	_, err = loc.SelectOption(playwright.SelectOptionValues{Labels: &[]string{label}})
	return err
}

// pickOption applies the matching rule to a native select: exact label
// first, then case-insensitive substring.
func pickOption(opts []string, want string) (string, bool) {
	for _, o := range opts {
		if strings.TrimSpace(o) == want {
			return o, true
		}
	}
	for _, o := range opts {
		if strings.Contains(strings.ToLower(o), strings.ToLower(want)) {
			return o, true
		}
	}
	return "", false
}

// popup clicks a link that opens a tab, checks the tab's URL, closes it.
func (r *runner) popup(root playwright.Locator, a scenario.Action) error {
	link, err := r.find(r.timeout(), fmt.Sprintf("link %q", a.Link), r.byText(root, a.Link)...)
	if err != nil {
		return err
	}
	if err := r.travel(link); err != nil {
		return err
	}
	tab, err := r.page.ExpectPopup(func() error { return link.Click() })
	if err != nil {
		return err
	}
	defer tab.Close()
	deadline := time.Now().Add(r.timeout())
	for !strings.Contains(tab.URL(), a.URLContains) {
		if !time.Now().Before(deadline) {
			return fmt.Errorf("the new tab is at %q, which does not contain %q", tab.URL(), a.URLContains)
		}
		time.Sleep(pollEvery)
	}
	return nil
}

// see asserts that every text is visible, in the page or in one of its
// frames (a mail catcher renders the message body in an iframe). Frames are
// listed again on each poll, as an iframe may load after the step's actions.
func (r *runner) see(texts []string) error {
	for _, t := range texts {
		deadline := time.Now().Add(r.timeout())
		for {
			var cands []playwright.Locator
			for _, f := range r.page.Frames() {
				cands = append(cands, r.byText(f.Locator("body"), t)...)
			}
			if _, ok := r.firstVisible(0, cands...); ok {
				break
			}
			if !time.Now().Before(deadline) {
				return fmt.Errorf("see: nothing visible matched %q within %s", t, r.timeout())
			}
			time.Sleep(pollEvery)
		}
	}
	return nil
}

// StepError names the step, and the action when one failed.
type StepError struct {
	Step    int
	Caption string
	Action  string
	Err     error
}

func (e *StepError) Error() string {
	if e.Action == "" {
		return fmt.Sprintf("step %d (%s): %v", e.Step, e.Caption, e.Err)
	}
	return fmt.Sprintf("step %d (%s): %s: %v", e.Step, e.Caption, e.Action, e.Err)
}

func (e *StepError) Unwrap() error { return e.Err }

// runStep plays the actions of a step, then asserts its see texts.
func (r *runner) runStep(n int, st scenario.Step) error {
	for _, a := range st.Do {
		if err := r.do(a); err != nil {
			return &StepError{Step: n, Caption: st.Caption, Action: a.String(), Err: err}
		}
		r.pause(r.gesture(afterAction))
	}
	if err := r.see(st.See); err != nil {
		return &StepError{Step: n, Caption: st.Caption, Err: err}
	}
	return nil
}
