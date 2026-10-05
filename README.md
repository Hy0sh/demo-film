# demo-film

`demo-film` films a demo of a web app, headless, from a declarative YAML
scenario, and outputs an mp4 with a caption band under the page plus a
chapter list. The scenario is data in a closed vocabulary, never code: the
same scenario films the same video, and a reviewer can read it.

For each step the band first shows the caption ("what I do"), the actions
play with a visible cursor and visible typing, the step's `see` texts are
asserted on screen, then the band adds "You should see: ..." and holds.

## Requirements

- Go, to install the binary.
- [`ffmpeg`](https://ffmpeg.org) on `PATH` (`brew install ffmpeg`,
  `apt install ffmpeg`). Only `film` needs it. Captions are rendered as
  images by the browser and overlaid by ffmpeg, so it needs neither libass
  nor drawtext.
- The Playwright driver and Chromium, installed by `demo-film install`.

`rehearse` and `film` check these first and print the exact command to run
when one is missing.

## Install

```sh
go install github.com/Hy0sh/demo-film/cmd/demo-film@latest
demo-film install        # the Playwright driver and Chromium, nothing else
```

Shell completion (bash/zsh/fish/powershell) comes from Cobra:
`demo-film completion zsh --help`.

## Commands

| Command | Does |
|---|---|
| `demo-film check <scenario.yaml>` | validates the scenario, no browser; exit 1 with every problem listed |
| `demo-film rehearse <scenario.yaml> [-o dir]` | plays every step with no pause and no video, asserting `see`; on failure exit 1, names the step and writes `rehearse-fail-step<N>.png` (full page) in `dir` (default: the current directory) |
| `demo-film film <scenario.yaml> -o <dir>` | records `<dir>/demo.mp4` and `<dir>/chapters.md`; a failing step aborts and writes nothing |
| `demo-film install` | installs the Playwright driver and Chromium |
| `demo-film --version` | prints the version |

Rehearse first, film when it is green. Pass the full scenario through
`check` while writing it.

## Scenario

```yaml
title: string
base_url: http://host:port            # relative `open` paths resolve against it
viewport: {width: 1440, height: 900}  # optional, this is the default; both even
locale: fr                            # optional, see below
hide: ["css selector", ...]           # optional, see below
timeout: 15                           # optional, seconds per action
speed: 1                              # optional, 0.25 to 4: gestures only (0.5 = twice as slow)
labels:                               # optional, English defaults shown
  step: Step                          #   "Step 2/5"
  check: check                        #   "Step 2/5 · check E3"
  see: "You should see:"
steps:
  - caption: string                   # shown BEFORE the actions
    check: string                     # optional tag of the acceptance point, e.g. "E3"
    do: [action, ...]
    see: ["text", ...]                # must be visible at the end of the step, else the run fails
    expect: string                    # shown AFTER the actions
```

Unknown keys are errors. `examples/shop.yaml` is a complete scenario.

- `locale` sets `localStorage.i18nextLng` before any page script runs.
- `hide` injects `display: none !important` for those selectors during
  filming (dev toolbars, debug overlays).
- `labels` lets a demo speak another language without hardcoding it, e.g.
  `{step: "Étape", check: "vérifie", see: "Tu dois voir :"}`.

### Actions

Each item of `do` is a map with one verb. Every pointer action first moves
the visible cursor smoothly to the centre of the element, then acts; typing
is visible, character by character.

| Action | Meaning |
|---|---|
| `open: url-or-path` | go to a page; see the rules below |
| `menu: [Parent, Child]` | click the navigation entry (link or button, by accessible name); if the child is not visible, click the parent first |
| `click: "Visible text"` | click the element showing that text |
| `click: {role: button, name: "Save"}` | click by role and accessible name |
| `click: {row: "text in a row", button: "Edit"}` | click the button, by accessible name, in the first table row containing the text |
| `click: {row: "text in a row", button: {nth: -2}}` | same, by index among the row's buttons (negative counts from the end), for icon-only buttons |
| `fill: {field: "Email", value: "..."}` | type in the field found by label or placeholder; `field: password` is the password input; `field: 2` is the 2nd visible text control (1-based) |
| `select: {field: "Category", option: "Lighting"}` | native `<select>`, by label or by rank (among the page's selects); option by visible label |
| `press: Enter` | press a key |
| `hover: "Visible text"` | move the cursor onto the element |
| `wait: "text"` | wait until the text is visible |
| `popup: {click: "Docs", url_contains: "/docs"}` | click a link that opens a new tab, assert the tab's URL contains the string, close the tab |
| `confirm: "Discard"` | click a button in the most recently opened dialog |

`within: dialog` is a modifier key allowed on `click`, `fill`, `select`,
`hover` and `wait`; it scopes the lookup to the last open `[role=dialog]`:

```yaml
- click: Next
  within: dialog
```

**Text matching**: exact visible text first, then case-insensitive
substring. Only visible elements match. A failure names the step, the
action and what was looked for. Each action gets `timeout` seconds
(default 15).

### Timing

Nothing is paused by hand. Per step, the caption is read for 60 ms per
character (at least 2.5 s), and the "you should see" state is held for 60
ms per character (at least 4 s); both capped at 9 s. The same scenario
always films the same video.

Gestures are paced for a human eye: the cursor travels to its target in
0.7 s with a smooth start and stop, rests 0.6 s before the click, and the
effect stays 0.7 s before the next action; typing takes 70 ms per
character. `speed` scales these gestures only, never the captions.

## Rules enforced by `check`

They are validation only, no browser needed, and they exist so a film cannot
quietly lie.

- **Schema.** Unknown keys, unknown verbs, malformed actions are errors; a
  scenario needs a title, a `base_url` and steps; each step needs a caption,
  at least one action and an `expect`. Viewport dimensions must be even
  (H.264 requires it).
- **A step with `check` needs a non-empty `see`.** An acceptance point that
  nothing asserts on screen is a claim, not a proof.
- **`open` is allowed in step 1, or to a different origin than the current
  page** (a mail catcher, say). A same-origin `open` after step 1 reloads
  the app: on a single-page app the video shows a blank page and the demo no
  longer shows what a user does. Navigate through the UI (`menu`, `click`).
- **The last action of a step may not be a `click` on a forward button**
  named `next`, `suivant`, `continue` or `continuer` (case-insensitive,
  matched on the whole name). The "you should see" caption appears after the
  last action: if that action only moves the flow forward, the caption
  describes the screen the viewer has just left. Let the next step open with
  that click, so the caption and the screen match.

## Output

`demo.mp4` is H.264 yuv420p with `faststart`, as wide as the viewport and
150 px taller than it: the page is untouched at the top, the dark caption
band (accent top border, "Step N/M · check X", caption, then "You should
see: ...") sits under it. `chapters.md` is a table with step, check,
mm:ss, caption and expect. The raw recording and the caption images are
removed.

## Development

```sh
go test -short ./...     # unit tests only
demo-film install        # once, for the integration tests
go test ./...            # + a real browser and ffmpeg on a tiny embedded app
```

The integration tests skip themselves when ffmpeg or the browser is missing.
