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
		{"short caption hits the read floor", ReadTime("Hi"), 2500 * time.Millisecond},
		{"short expect hits the hold floor", HoldTime("Hi"), 4 * time.Second},
		{"read scales with length", ReadTime(strings.Repeat("a", 50)), 3 * time.Second},
		{"hold scales with length", HoldTime(strings.Repeat("a", 100)), 6 * time.Second},
		{"read is capped", ReadTime(strings.Repeat("a", 1000)), 9 * time.Second},
		{"hold is capped", HoldTime(strings.Repeat("a", 1000)), 9 * time.Second},
		{"length counts characters, not bytes", ReadTime(strings.Repeat("é", 50)), 3 * time.Second},
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
