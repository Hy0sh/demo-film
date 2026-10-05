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

func clamp(text string, floor time.Duration) time.Duration {
	d := time.Duration(utf8.RuneCountInString(text)) * perChar
	return min(max(d, floor), maxWait)
}

// ReadTime is how long a caption stays alone before the actions start.
func ReadTime(caption string) time.Duration { return clamp(caption, minRead) }

// HoldTime is how long the "you should see" state stays after the actions.
func HoldTime(expect string) time.Duration { return clamp(expect, minHold) }

// Clock renders a duration as m:ss.
func Clock(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
