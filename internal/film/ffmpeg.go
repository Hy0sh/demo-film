package film

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Window is the time span, in seconds, during which a caption image shows.
type Window struct{ Start, End float64 }

// Filter pads the page video with the band area, then overlays each caption
// image (input 1, 2...) under the page during its window. The result is the
// stream labelled [v<len(windows)>]. This ffmpeg-only route needs neither
// libass nor drawtext: the captions are rendered as images by the browser.
func Filter(width, height int, windows []Window) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[0:v]pad=%d:%d:0:0:color=0x%s[v0]", width, height+BandHeight, strings.TrimPrefix(bandBackground, "#"))
	for k, w := range windows {
		fmt.Fprintf(&b, ";[v%d][%d:v]overlay=0:%d:enable='between(t,%.2f,%.2f)'[v%d]", k, k+1, height, w.Start, w.End, k+1)
	}
	return b.String()
}

// assemble writes the mp4: H.264, yuv420p, faststart. The raw video is cut
// before skip (the pre-roll), so its time 0 is the first step's caption.
func assemble(raw string, skip time.Duration, pngs []string, filter, out string, last int) error {
	args := []string{"-y", "-loglevel", "error", "-ss", fmt.Sprintf("%.3f", skip.Seconds()), "-i", raw}
	for _, p := range pngs {
		args = append(args, "-i", p)
	}
	args = append(args, "-filter_complex", filter, "-map", fmt.Sprintf("[v%d]", last),
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", out)
	if msg, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, msg)
	}
	return nil
}
