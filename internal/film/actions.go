package film

import (
	_ "embed"
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

// runner plays the actions of a scenario on a page. Paced, as when filming
// or in a paced rehearsal, it adds the pauses and the visible typing a
// viewer needs; a plain rehearsal skips them.
type runner struct {
	s     *scenario.Scenario
	page  playwright.Page
	paced bool
	x, y  float64 // last cursor position, where the next travel starts
	cuts  []span  // the waits marked cut, removed from the video
}

// span is a stretch of wall-clock time.
type span struct{ from, to time.Time }

// cutTotal is the time the cuts so far remove from the video.
func (r *runner) cutTotal() time.Duration {
	var d time.Duration
	for _, c := range r.cuts {
		d += c.to.Sub(c.from)
	}
	return d
}

// cardTime is how long the transition card shows on each side of a cut.
const cardTime = 1200 * time.Millisecond

// minCut is the shortest wait worth a cut: below it, "⏩ 0:00 later" would
// announce a jump the viewer cannot notice.
const minCut = time.Second

// cutWait plays a wait marked cut: a transition card covers the page with a
// spinner, the wait happens under it and is cut out, then the card names the
// time skipped ("⏩ 2:14 later") and fades onto the result. The viewer sees
// that the film jumped: a cut never passes for an instant task.
//
// A wait with nothing to wait for shows no card: when the condition already
// holds (a script off camera finished first), the film goes on; when it
// comes true under minCut, the card leaves without a cut nor a "0:00 later".
func (r *runner) cutWait(timeout time.Duration, find func(time.Duration) error) error {
	if find(0) == nil {
		return nil
	}
	if _, err := r.page.Evaluate(cardJS, ""); err != nil {
		return err
	}
	r.pause(r.gesture(cardTime))
	from := time.Now()
	if err := find(timeout); err != nil {
		return err
	}
	if time.Since(from) < minCut {
		_, err := r.page.Evaluate(cardJS, nil)
		return err
	}
	r.cuts = append(r.cuts, span{from, time.Now()})
	if _, err := r.page.Evaluate(cardJS, "⏩ "+Clock(time.Since(from))+" "+r.s.Labels.Later); err != nil {
		return err
	}
	r.pause(max(r.gesture(cardTime), minShown))
	_, err := r.page.Evaluate(cardJS, nil)
	return err
}

//go:embed card.js
var cardJS string

func (r *runner) timeout() time.Duration { return time.Duration(r.s.Timeout) * time.Second }

func (r *runner) pause(d time.Duration) {
	if r.paced {
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
		return nil, fmt.Errorf("nothing visible matched %s within %s%s", what, timeout, r.suggest())
	}
	return loc, nil
}

//go:embed names.js
var namesJS string

// maxSuggestions keeps a failure message readable on a busy screen.
const maxSuggestions = 40

// suggest lists the names a scenario could aim at on the current screen, so
// a wrong label is fixed from the error alone, without exploring the app.
func (r *runner) suggest() string {
	v, err := r.page.Evaluate(namesJS)
	if err != nil {
		return ""
	}
	list, _ := v.([]interface{})
	var names []string
	for _, n := range list {
		if s, ok := n.(string); ok {
			names = append(names, fmt.Sprintf("%q", s))
		}
	}
	if len(names) == 0 {
		return ""
	}
	more := ""
	if len(names) > maxSuggestions {
		more = fmt.Sprintf(" (+%d more)", len(names)-maxSuggestions)
		names = names[:maxSuggestions]
	}
	return "; visible on this screen: " + strings.Join(names, ", ") + more
}

func (r *runner) byText(root playwright.Locator, text string) []playwright.Locator {
	return []playwright.Locator{
		root.GetByText(match(text, true), playwright.LocatorGetByTextOptions{Exact: playwright.Bool(true)}),
		root.GetByText(match(text, false), playwright.LocatorGetByTextOptions{Exact: playwright.Bool(false)}),
	}
}

func (r *runner) byName(root playwright.Locator, role, name string) []playwright.Locator {
	var out []playwright.Locator
	for _, exact := range []bool{true, false} {
		out = append(out, root.GetByRole(playwright.AriaRole(role), playwright.LocatorGetByRoleOptions{Name: match(name, exact), Exact: playwright.Bool(exact)}))
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
	if r.paced {
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

//go:embed picker_value.js
var pickerValueJS string

// nativePicker are the input types whose value is picked, not typed.
var nativePicker = map[string]bool{"date": true, "month": true, "week": true, "time": true, "datetime-local": true, "color": true, "range": true}

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
		byLabel := root.GetByLabel(match(f.Label, exact), playwright.LocatorGetByLabelOptions{Exact: playwright.Bool(exact)})
		if selects {
			out = append(out, byLabel)
			continue
		}
		out = append(out, byLabel.Or(root.GetByPlaceholder(match(f.Label, exact), playwright.LocatorGetByPlaceholderOptions{Exact: playwright.Bool(exact)})))
	}
	return out
}

// do plays one action.
func (r *runner) do(a scenario.Action) error {
	root := r.root(a)
	to := r.timeout()
	if a.Timeout > 0 {
		to = time.Duration(a.Timeout) * time.Second
	}
	switch a.Kind {
	case scenario.Open:
		url, err := scenario.ResolveURL(r.s.BaseURL, a.Text)
		if err != nil {
			return err
		}
		if _, err = r.page.Goto(url); err != nil || r.s.Terminal == nil {
			return err
		}
		_, err = r.page.WaitForFunction(promptJS, nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(float64(to.Milliseconds()))})
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
		// A native picker (date, month, time...) takes no typed characters:
		// its value is set at once, as rehearse does for every field.
		kind, _ := loc.GetAttribute("type")
		if nativePicker[kind] {
			// An app may reject the value it is given (a controlled input):
			// a picker that kept something else is a failure, not a green.
			if err := loc.Fill(a.Value); err != nil {
				return err
			}
			// Compared with what the browser makes of the value given, so
			// "#AABBCC" or "08:00:00" pass as the "#aabbcc" or "08:00" kept.
			want, err := r.page.Evaluate(pickerValueJS, []string{kind, a.Value})
			if err != nil {
				return err
			}
			if got, err := loc.InputValue(); err != nil || got != want {
				return fmt.Errorf("the %s field holds %q, not %q (the app refused it, or the value is not in the input's format: 2026-10-06, 2026-10, 08:00, 2026-10-06T08:00, #aabbcc)", kind, got, a.Value)
			}
			return nil
		}
		if !r.paced {
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

	case scenario.Type:
		if !r.paced {
			return r.page.Keyboard().Type(a.Text)
		}
		return r.page.Keyboard().Type(a.Text, playwright.KeyboardTypeOptions{Delay: playwright.Float(typingDelay / r.s.Speed)})

	case scenario.Hover:
		loc, err := r.find(to, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
		if err != nil {
			return err
		}
		return r.travel(loc)

	case scenario.Wait:
		find := func(timeout time.Duration) error {
			switch a.Until {
			case scenario.Gone:
				return r.gone(timeout, a.Text, r.byText(root, a.Text)...)
			case scenario.Enabled:
				return r.enabled(timeout, a.Text, append(r.byName(root, "button", a.Text), r.byText(root, a.Text)...)...)
			}
			_, err := r.find(timeout, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
			return err
		}
		if !a.Cut || !r.paced {
			return find(to)
		}
		return r.cutWait(to, find)

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
		row := root.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: match(a.Row, false)})
		if a.Nth != nil {
			return r.find(to, fmt.Sprintf("button #%d in the row %q", *a.Nth, a.Row),
				row.First().Locator("button").Locator("visible=true").Nth(*a.Nth))
		}
		return r.find(to, fmt.Sprintf("button %q in the row %q", a.Button, a.Row),
			r.byName(row.First(), "button", a.Button)...)
	case a.Role != "":
		return r.pick(to, a.Match, fmt.Sprintf("%s %q", a.Role, a.Name), r.byName(root, a.Role, a.Name)...)
	}
	return r.pick(to, a.Match, fmt.Sprintf("%q", a.Text), r.byText(root, a.Text)...)
}

// gone waits until no candidate shows a visible element any more: a toast
// that covers a button, a spinner, or a text that must not be there.
func (r *runner) gone(timeout time.Duration, text string, cands ...playwright.Locator) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := r.firstVisible(0, cands...); !ok {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("%q is still visible after %s", text, timeout)
		}
		time.Sleep(pollEvery)
	}
}

// enabled waits until the first visible candidate, a button by name before
// a text, is no longer disabled: a "Continue" greyed until a list loads.
func (r *runner) enabled(timeout time.Duration, text string, cands ...playwright.Locator) error {
	loc, err := r.find(timeout, fmt.Sprintf("%q", text), cands...)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		if ok, err := loc.IsEnabled(); err == nil && ok {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("%q is still disabled after %s", text, timeout)
		}
		time.Sleep(pollEvery)
	}
}

// pick is find, or with nth, the nth visible match (from 0, negative from
// the end) of the first candidate that has any: several buttons may share a
// text, one per day or per card. Out of range names how many match.
func (r *runner) pick(timeout time.Duration, nth *int, what string, cands ...playwright.Locator) (playwright.Locator, error) {
	if nth == nil {
		return r.find(timeout, what, cands...)
	}
	deadline := time.Now().Add(timeout)
	found := 0
	for {
		for _, c := range cands {
			vis := c.Locator("visible=true")
			n, err := vis.Count()
			if err != nil || n == 0 {
				continue
			}
			found = n
			if i := *nth; i < n && i >= -n {
				return vis.Nth(i), nil
			}
			break
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(pollEvery)
	}
	if found == 0 {
		return nil, fmt.Errorf("nothing visible matched %s within %s%s", what, timeout, r.suggest())
	}
	return nil, fmt.Errorf("nth %d is out of range: %d visible elements match %s", *nth, found, what)
}

// entry is a navigation entry: a link or a button, by exact accessible name.
// No substring fallback: a hidden "Settings" child would otherwise match a
// visible "General settings" entry elsewhere in the menu.
func (r *runner) entry(root playwright.Locator, name string) []playwright.Locator {
	opts := playwright.LocatorGetByRoleOptions{Name: match(name, true), Exact: playwright.Bool(true)}
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
	// Options often arrive after the select itself: look again until the
	// timeout before saying the option is missing.
	deadline := time.Now().Add(r.timeout())
	for {
		opts, err := loc.Locator("option").AllInnerTexts()
		if err != nil {
			return err
		}
		if label, ok := pickOption(opts, a.Option); ok {
			_, err = loc.SelectOption(playwright.SelectOptionValues{Labels: &[]string{label}})
			return err
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("select %s has no option %q within %s (options: %q)", a.Field, a.Option, r.timeout(), opts)
		}
		time.Sleep(pollEvery)
	}
}

// pickOption applies the matching rule to a native select: exact label
// first, then case-insensitive substring, any spaces alike.
func pickOption(opts []string, want string) (string, bool) {
	want = sameSpacing(want)
	for _, o := range opts {
		if sameSpacing(o) == want {
			return o, true
		}
	}
	for _, o := range opts {
		if strings.Contains(strings.ToLower(sameSpacing(o)), strings.ToLower(want)) {
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
