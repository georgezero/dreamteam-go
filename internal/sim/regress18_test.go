package sim_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review finding 1: extension minutes are validated as given,
// before any conversion; a huge value must not wrap into a negative
// duration. Invalid actions change nothing.
func TestRegress18ExtensionMinutesRange(t *testing.T) {
	for _, c := range []struct {
		minutes int64
		ok      bool
	}{
		{0, false}, {-1, false}, {-60, false}, {math.MinInt64, false},
		{241, false}, {math.MaxInt64, false}, {4611686018427387903, false},
		{1, true}, {240, false}, // Jordan (desk) 16:15 + 240 = 20:15 > cap 19:45: valid minutes, other rule
		{210, true},
	} {
		w, d0 := miniDay(t, 1)
		if err := runTo(w, at(d0, "07:00")); err != nil {
			t.Fatal(err)
		}
		before := w.Observe(0)
		preview := w.Validate(extend("E01", c.minutes))
		r := w.Apply([]api.Action{extend("E01", c.minutes)})[0]
		after := w.Observe(0)
		rangeErr := r.Code == "invalid_minutes"
		wantRange := c.minutes < 1 || c.minutes > 240
		if r.OK != c.ok || preview.OK != c.ok || rangeErr != wantRange {
			t.Errorf("minutes %d: apply %+v, preview %+v", c.minutes, r, preview)
		}
		if !r.OK {
			before.Actions, after.Actions = api.ActionLog{}, api.ActionLog{}
			before.Today.Actions, after.Today.Actions = api.ActionCounts{}, api.ActionCounts{}
			if !reflect.DeepEqual(before, after) {
				t.Errorf("minutes %d: a rejected extension changed the state", c.minutes)
			}
		} else if e := staffID(after, "Jordan"); e.ApprovedMin != c.minutes {
			t.Errorf("minutes %d: approved %d", c.minutes, e.ApprovedMin)
		}
	}
	_ = simtime.Minute
}
