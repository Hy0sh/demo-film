# Changelog

What changed between published versions, and why. Versions follow
[semantic versioning](https://semver.org): while the major stays at 0, a minor
bump carries new commands or new behaviour, a patch bump carries fixes.

## [Unreleased]

### Fixed

- `fill` on a native date, month, week, time, datetime-local, colour or
  range input failed when filming while the rehearsal passed: such a field
  takes no typed characters, its value is now set at once. The step now
  fails when such a field does not hold the value given (an app rejecting
  it), instead of passing.

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

[Unreleased]: https://github.com/Hy0sh/demo-film/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/Hy0sh/demo-film/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Hy0sh/demo-film/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Hy0sh/demo-film/releases/tag/v0.1.0
