package simtime

import (
	"fmt"
	"testing"
)

// The allocation-light formatters produce exactly the text of the
// fmt-based versions they replaced (log text, and so the log hash, is
// unchanged).
func TestFormattersMatchFmt(t *testing.T) {
	ref := func(d Duration) string {
		neg := d < 0
		if neg {
			d = -d
		}
		var s string
		switch {
		case d < Minute:
			s = fmt.Sprintf("%ds", int64(d))
		case d < Hour:
			s = fmt.Sprintf("%dm", int64(d/Minute))
		case d < Day:
			h, m := int64(d/Hour), int64(d%Hour/Minute)
			if m == 0 {
				s = fmt.Sprintf("%dh", h)
			} else {
				s = fmt.Sprintf("%dh %02dm", h, m)
			}
		default:
			days, h := int64(d/Day), int64(d%Day/Hour)
			if h == 0 {
				s = fmt.Sprintf("%dd", days)
			} else {
				s = fmt.Sprintf("%dd %dh", days, h)
			}
		}
		if neg {
			return "-" + s
		}
		return s
	}
	for d := -3 * Day; d <= 40*Day; d += 7 {
		if got, want := d.String(), ref(d); got != want {
			t.Fatalf("Duration(%d) = %q, want %q", d, got, want)
		}
		if got, want := FormatClock(d), fmt.Sprintf("%02d:%02d", int64(floorMod(int64(d), int64(Day))/int64(Hour)), int64(floorMod(int64(d), int64(Day))%int64(Hour)/int64(Minute))); got != want {
			t.Fatalf("FormatClock(%d) = %q, want %q", d, got, want)
		}
	}
	cal := NewCalendar(Date{1999, 12, 25})
	for tm := Time(-5 * Day); tm < Time(800*Day); tm += 7919 {
		d := cal.Date(tm)
		cl := ClockOf(tm)
		want := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d", d.Year, d.Month, d.Day, int64(cl/Hour), int64(cl%Hour/Minute), int64(cl%Minute))
		if got := cal.ISO(tm); got != want {
			t.Fatalf("ISO(%d) = %q, want %q", tm, got, want)
		}
	}
}
