# Changelog

What changed between published versions, and why. Versions follow
[semantic versioning](https://semver.org): while the major stays at 0, a minor
bump carries new commands or new behaviour, a patch bump carries fixes.

## [Unreleased]

## [0.6.0] - 2026-10-07

### Changed

- `locale` now sets the words of the caption band and the cards too: `fr`
  gives "Étape", "vérifie", "À l'écran :" and "plus tard", and `labels`
  only overrides them. A French film whose labels left out `later` showed
  "⏩ 1:23 later" in English. A language without words of its own still
  gets English.
- A `cut` wait whose condition already holds shows no card any more, and
  one met within a second leaves its card without a cut: a script finished
  off camera produced a "⏩ 0:00 later" card.

## [0.5.0] - 2026-10-07

### Added

- `wait: {gone: "text"}` waits until nothing visible shows the text (a
  toast over a button, a spinner) and proves an absence; `wait: {enabled:
  "text"}` waits until a control is no longer disabled. Both take `within`,
  `timeout` and `cut`. They replace the hovers scenarios used to buy time,
  which showed as stray cursor moves in the video. A toast pauses while
  hovered, and a rehearsal's cursor lands on it at once: wait for it to go.
- `rehearse --paced` plays at the take's pace (captions read and held,
  cursor travel, typing) without video: slower, but what depends on time
  behaves as in the take.

### Fixed

- A space in a scenario text now matches any white space on the page,
  no-break spaces included: a French placeholder "ex : Natation", written
  with U+202F, was not found from a scenario typed with a plain space,
  though the list of visible names showed it. Applies to texts, roles and
  names, labels, placeholders, menu entries, table rows and select options.
- `select` failed at once when its options had not loaded yet (fetched
  after the select showed); it now looks for the option until the timeout.

## [0.4.0] - 2026-10-06

### Added

- `nth` on `click` by text or by role and name picks among several visible
  matches (one slot per day, one "Actions" button per card), from 0,
  negative from the end; out of range, the error says how many match.
- `cut: true` on `wait` removes a long task from the video: it jumps to the
  moment the awaited text shows, under a transition card that reads
  "⏩ 2:14 later" before fading onto the result. `timeout: N` on `wait`
  gives that task more than the scenario's timeout. The card's word comes
  from the new `labels.later`.
- `terminal` films a shell instead of a web app, served in the browser by
  ttyd (a new requirement, for terminal demos only), and the new `type`
  action types into it.
- `join` assembles several films into one, a title card before each and
  their chapters merged with shifted times. `film` now records the speed in
  the mp4, which `join` needs: films made before cannot be joined.
- `watermark` signs a video with an image or a line of text, in a corner of
  the page area, at a chosen opacity.

### Fixed

- `fill` on a native date, month, week, time, datetime-local, colour or
  range input failed when filming while the rehearsal passed: such a field
  takes no typed characters, its value is now set at once. The step now
  fails when such a field does not hold the value given (an app rejecting
  it), instead of passing.
- `locale` only set i18next: native date and month inputs, `Intl` and
  `Accept-Language` stayed in English. It now sets the browser's language
  too.

## [0.3.0] - 2026-10-06

### Changed

- `speed` now scales the whole video: caption read and hold times follow it
  as well as the gestures (x2 = about twice as fast, x0.5 twice as slow),
  with a 1.2 s floor so a caption stays readable. At `speed: 1` (the
  default) nothing changes.

## [0.2.0] - 2026-10-05

### Added

- A lookup that finds nothing now lists the names visible on the screen
  (links, buttons, tabs, fields, labels) in its error: a wrong label is fixed
  from the error alone, without exploring the app in a browser.

## [0.1.0] - 2026-10-05

First version: film a web app demo headless from a YAML scenario.

### Added

- `demo-film check`: validates a scenario without a browser (strict schema,
  a `see` for every `check`, no same-origin `open` after step 1, no step
  ending on a forward button).
- `demo-film rehearse`: plays every step with no pause and no video,
  asserting each `see`; a failure names the step and leaves a full-page
  screenshot.
- `demo-film film`: records `demo.mp4` (page plus a caption band, H.264
  yuv420p) and `chapters.md`, with read and hold times computed from the
  text length.
- `demo-film install`: installs the Playwright driver and Chromium.
- A closed vocabulary of actions: `open`, `menu`, `click`, `fill`, `select`,
  `press`, `hover`, `wait`, `popup`, `confirm`, and the `within: dialog`
  modifier. Menu entries match by exact name; `see` texts are also looked
  for in the page's frames (a mail catcher's message body).
- Gestures paced for a human eye (cursor travel, rest before the click,
  pause after each action, visible typing), scaled by the scenario's
  `speed` (0.25 to 4).
- The video starts once the app shows: when step 1 starts with `open`, the
  page loads off camera and the blank page is cut.

[Unreleased]: https://github.com/Hy0sh/demo-film/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/Hy0sh/demo-film/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/Hy0sh/demo-film/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Hy0sh/demo-film/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Hy0sh/demo-film/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Hy0sh/demo-film/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Hy0sh/demo-film/releases/tag/v0.1.0
