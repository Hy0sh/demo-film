# Changelog

What changed between published versions, and why. Versions follow
[semantic versioning](https://semver.org): while the major stays at 0, a minor
bump carries new commands or new behaviour, a patch bump carries fixes.

## [Unreleased]

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
  modifier.

[Unreleased]: https://github.com/Hy0sh/demo-film/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Hy0sh/demo-film/releases/tag/v0.1.0
