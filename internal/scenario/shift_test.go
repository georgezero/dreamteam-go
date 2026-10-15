package scenario

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/scenarios"
)

func compileOverlay(t *testing.T, overlay string) (*sim.Config, error) {
	t.Helper()
	base, _ := scenarios.FS.ReadFile("base/established.json")
	fsys := fstest.MapFS{
		"base/established.json": {Data: base},
		"o.json":                {Data: []byte(`{"extends": "base/established.json", ` + strings.TrimPrefix(overlay, "{"))},
	}
	f, err := Load(fsys, "o.json")
	if err != nil {
		t.Fatalf("%s: load: %v", overlay, err)
	}
	return Compile(f)
}

// V0.2 Step 0C: maxShiftHours caps a shift's length (and its holdover), not
// its start. A short shift may start after noon; it must end by accounting
// close, and its cap is the earlier of start + maxShiftHours and 23:59:59.
func TestShortLateShifts(t *testing.T) {
	tech := func(start, end string) string {
		return `{"employees": [{"id": "E09", "name": "Jamie", "role": "technologist", "skills": ["XR"],
			"pay": {"type": "hourly", "rateCents": 3800},
			"roster": {"Tue": {"start": "` + start + `", "end": "` + end + `", "assignment": "R1"}}}]}`
	}
	for _, ok := range []string{
		tech("16:00", "20:00"),
		tech("17:00", "20:00"),
		tech("11:30", "20:00"),
		tech("19:00", "23:59"),
		// Rejected before Step 0C ("start + maxShiftHours must end by
		// accounting close"): a 13:00–20:00 shift under a 16-hour cap.
		`{"policies": {"maxShiftHours": 16}, "employees": [{"id": "E08", "roster": {"Tue": {"start": "13:00"}}}]}`,
	} {
		if _, err := compileOverlay(t, ok); err != nil {
			t.Errorf("%s: want valid, got %v", ok, err)
		}
	}
	for _, c := range []struct{ overlay, path, msg string }{
		{tech("06:00", "19:00"), "employees[8].roster.Tue", "longer than maxShiftHours"},
		{tech("20:00", "24:00"), "employees[8].roster.Tue", "accounting close"},
		{`{"policies": {"maxShiftHours": 4}, "employees": [{"id": "E09", "name": "Jamie", "role": "technologist", "skills": ["XR"],
			"pay": {"type": "hourly", "rateCents": 3800}, "roster": {"Tue": {"start": "15:00", "end": "20:00", "assignment": "R1"}}}]}`,
			"employees[8].roster.Tue", "longer than maxShiftHours"},
	} {
		_, err := compileOverlay(t, c.overlay)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want validation error, got %v", c.overlay, err)
			continue
		}
		found := false
		for _, is := range ve.Issues {
			found = found || is.Path == c.path && strings.Contains(is.Msg, c.msg)
		}
		if !found {
			t.Errorf("%s: no %q at %s in\n%v", c.overlay, c.msg, c.path, err)
		}
	}
}

func TestShiftCap(t *testing.T) {
	p := sim.Policies{MaxShift: 12 * simtime.Hour}
	for _, c := range []struct{ from, want string }{
		{"07:45", "19:45"}, {"11:30", "23:30"}, {"11:59", "23:59:00"}, {"12:00", "23:59:59"}, {"16:00", "23:59:59"},
	} {
		from, _ := simtime.ParseClock(c.from)
		want, _ := simtime.ParseClock(c.want)
		if got := p.ShiftCap(from); got != want {
			t.Errorf("cap from %s = %v, want %s", c.from, got, c.want)
		}
	}
}
