package film

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Window is the time span, in seconds, during which a caption image shows.
type Window struct{ Start, End float64 }

// Filter keeps the stretches of the page video in keep (all of it when keep
// is empty) and joins them, pads it with the band area, then overlays each
// caption image (input 1, 2...) under the page during its window. The
// result is the stream labelled [v<len(windows)>]. This ffmpeg-only route
// needs neither libass nor drawtext: the captions are rendered as images by
// the browser.
func Filter(width, height int, keep, windows []Window) string {
	var b strings.Builder
	src := "[0:v]"
	if len(keep) > 0 {
		fmt.Fprintf(&b, "[0:v]split=%d", len(keep))
		for i := range keep {
			fmt.Fprintf(&b, "[k%d]", i)
		}
		for i, k := range keep {
			fmt.Fprintf(&b, ";[k%d]trim=start=%.3f:end=%.3f,setpts=PTS-STARTPTS[s%d]", i, k.Start, k.End, i)
		}
		b.WriteString(";")
		for i := range keep {
			fmt.Fprintf(&b, "[s%d]", i)
		}
		fmt.Fprintf(&b, "concat=n=%d[cut];", len(keep))
		src = "[cut]"
	}
	fmt.Fprintf(&b, "%spad=%d:%d:0:0:color=0x%s[v0]", src, width, height+BandHeight, strings.TrimPrefix(bandBackground, "#"))
	for k, w := range windows {
		fmt.Fprintf(&b, ";[v%d][%d:v]overlay=0:%d:enable='between(t,%.2f,%.2f)'[v%d]", k, k+1, height, w.Start, w.End, k+1)
	}
	return b.String()
}

// watermarkMargin is the gap between the watermark and the page's edges.
const watermarkMargin = 24

// WatermarkFilter overlays input `in` (the watermark image) on the stream
// [v<last>] at that opacity, in a corner of the page area (height is the
// page's, so the band stays clear), into [v<last+1>].
func WatermarkFilter(in, last int, position string, opacity float64, height int) string {
	m := watermarkMargin
	x, y := fmt.Sprint(m), fmt.Sprint(m)
	if strings.HasSuffix(position, "right") {
		x = fmt.Sprintf("W-w-%d", m)
	}
	if strings.HasPrefix(position, "bottom") {
		y = fmt.Sprintf("%d-h-%d", height, m)
	}
	return fmt.Sprintf(";[%d:v]format=rgba,colorchannelmixer=aa=%.2f[wm];[v%d][wm]overlay=%s:%s[v%d]", in, opacity, last, x, y, last+1)
}

// speedTag is the mp4 metadata key holding the scenario's speed, which join
// needs to pace its title cards and to refuse mixed speeds.
const speedTag = "demo_film_speed"

// encodeArgs end every ffmpeg run that writes a film: H.264, yuv420p,
// faststart, and the speed tag.
func encodeArgs(speed float64, out string) []string {
	return []string{"-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart+use_metadata_tags",
		"-metadata", fmt.Sprintf("%s=%g", speedTag, speed), out}
}

func ffmpeg(args ...string) error {
	if msg, err := exec.Command("ffmpeg", append([]string{"-y", "-loglevel", "error"}, args...)...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, msg)
	}
	return nil
}

// assemble writes the mp4. The raw video is cut before skip (the pre-roll),
// so its time 0 is the first step's caption.
func assemble(raw string, skip time.Duration, pngs []string, filter, out string, last int, speed float64) error {
	args := []string{"-ss", fmt.Sprintf("%.3f", skip.Seconds()), "-i", raw}
	for _, p := range pngs {
		args = append(args, "-i", p)
	}
	args = append(args, "-filter_complex", filter, "-map", fmt.Sprintf("[v%d]", last))
	return ffmpeg(append(args, encodeArgs(speed, out)...)...)
}
