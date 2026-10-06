// Package film runs a scenario in a headless browser, and assembles the video.
package film

import (
	"fmt"
	"time"
	"unicode/utf8"
)

const perChar = 60 * time.Millisecond

const (
	minRead = 2500 * time.Millisecond
	minHold = 4 * time.Second
	maxWait = 9 * time.Second
)

// minShown is the least a caption stays on screen at any speed: under about
// a second a changing caption cannot be read.
const minShown = 1200 * time.Millisecond

// scaled is the time to read text at that speed: the length-based time,
// floored and capped as at speed 1, then divided by the speed, never under
// minShown.
func scaled(text string, floor time.Duration, speed float64) time.Duration {
	d := time.Duration(utf8.RuneCountInString(text)) * perChar
	d = min(max(d, floor), maxWait)
	return max(time.Duration(float64(d)/speed), minShown)
}

// ReadTime is how long a caption stays alone before the actions start.
func ReadTime(caption string, speed float64) time.Duration {
	return scaled(caption, minRead, speed)
}

// HoldTime is how long the "you should see" state stays after the actions.
func HoldTime(expect string, speed float64) time.Duration {
	return scaled(expect, minHold, speed)
}

// Clock renders a duration as m:ss.
func Clock(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
