package film

import (
	"strings"
	"testing"
	"time"
)

func TestReadAndHoldTimes(t *testing.T) {
	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"short caption hits the read floor", ReadTime("Hi", 1), 2500 * time.Millisecond},
		{"short expect hits the hold floor", HoldTime("Hi", 1), 4 * time.Second},
		{"read scales with length", ReadTime(strings.Repeat("a", 50), 1), 3 * time.Second},
		{"hold scales with length", HoldTime(strings.Repeat("a", 100), 1), 6 * time.Second},
		{"read is capped", ReadTime(strings.Repeat("a", 1000), 1), 9 * time.Second},
		{"hold is capped", HoldTime(strings.Repeat("a", 1000), 1), 9 * time.Second},
		{"length counts characters, not bytes", ReadTime(strings.Repeat("é", 50), 1), 3 * time.Second},
		{"short caption at x0.5", ReadTime("Hi", 0.5), 5 * time.Second},
		{"short caption at x2", ReadTime("Hi", 2), 1250 * time.Millisecond},
		{"short caption at x4 hits the readability floor", ReadTime("Hi", 4), 1200 * time.Millisecond},
		{"50 chars at x4 hits the readability floor", ReadTime(strings.Repeat("a", 50), 4), 1200 * time.Millisecond},
		{"short expect at x2", HoldTime("Hi", 2), 2 * time.Second},
		{"short expect at x4 hits the readability floor", HoldTime("Hi", 4), 1200 * time.Millisecond},
		{"hold scales with length at x2", HoldTime(strings.Repeat("a", 100), 2), 3 * time.Second},
		{"capped text at x2", ReadTime(strings.Repeat("a", 1000), 2), 4500 * time.Millisecond},
		{"capped text at x0.5 may exceed the cap", ReadTime(strings.Repeat("a", 1000), 0.5), 18 * time.Second},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestClock(t *testing.T) {
	if got := Clock(75 * time.Second); got != "1:15" {
		t.Errorf("Clock = %q", got)
	}
}
