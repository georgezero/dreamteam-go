// Package scenario loads, merges, validates and compiles scenario files
// (PLAN §8) into sim.Config. It is the only place that reads JSON scenarios.
package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Schema is the scenario format version.
const Schema = "dreamteam.scenario/v1"

// File is a scenario after extends/preset merging. Only the fields the
// current version uses are defined; unknown fields are rejected so typos do
// not pass silently. Units are in the key names.
type File struct {
	Schema           string                `json:"schema"`
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Family           string                `json:"family,omitempty"`
	Extends          string                `json:"extends,omitempty"`
	Preset           string                `json:"preset,omitempty"`
	StartDate        string                `json:"startDate"`
	StartTime        string                `json:"startTime"`
	WarmupDays       int64                 `json:"warmupDays"`
	RotaWeekA        string                `json:"rotaWeekA,omitempty"`
	OpeningCashCents int64                 `json:"openingCashCents"`
	Hours            map[string]*[2]string `json:"hours"`
	Rooms            []Room                `json:"rooms"`
	Employees        []Employee            `json:"employees"`
	Policies         Policies              `json:"policies"`
	Catalog          []Exam                `json:"catalog"`
	Templates        map[string][]Template `json:"templates"`
	Sources          []Source              `json:"referralSources"`
	Demand           Demand                `json:"demand"`
	Behavior         Behavior              `json:"behavior"`
	WalkIns          WalkIns               `json:"walkIns"`
	OpeningSchedule  []OpeningAppt         `json:"openingSchedule,omitempty"`
}

// Exam is a catalog entry. Internal codes, not CPT codes.
type Exam struct {
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	Modality       string   `json:"modality"`
	Category       string   `json:"category"` // slot category it books into, e.g. "MR_45"
	Contrast       bool     `json:"contrast"`
	WalkInEligible bool     `json:"walkInEligible"`
	RoomTimeSec    Quantile `json:"roomTimeSec"`
	TechExtraSec   Quantile `json:"techExtraSec"`
}

// Template is a group of weekdays sharing a room's slot list.
type Template struct {
	Days  []string   `json:"days"`
	Slots []SlotSpec `json:"slots"`
}

// SlotSpec is one slot, or Count back-to-back slots of the same length.
type SlotSpec struct {
	Start    string `json:"start"`
	LenMin   int64  `json:"lenMin"`
	Count    int64  `json:"count,omitempty"` // default 1
	Category string `json:"category"`
}

// Source is a referral source.
type Source struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name"`
	Kind                 string           `json:"kind"`
	BaseDailyMilliOrders int64            `json:"baseDailyMilliOrders"`
	PriorityMixBP        map[string]int64 `json:"priorityMixBP"`
	ExamMix              map[string]int64 `json:"examMix"`
}

// Quantile is a piecewise-linear inverse CDF: keys "p0" … "p100" (whole
// percents) mapped to values.
type Quantile map[string]int64

// Demand shapes order arrival.
type Demand struct {
	HourProfileBP   map[string]int64 `json:"hourProfileBP"` // "07" → share of the day's orders
	WeekdayFactorBP map[string]int64 `json:"weekdayFactorBP"`
	SeasonFactorBP  []int64          `json:"seasonFactorBP"` // Jan … Dec
	LeadTimeDays    Quantile         `json:"leadTimeDays"`
	WindowMixBP     map[string]int64 `json:"windowMixBP"`
}

// Behavior is patient behavior used by booking.
type Behavior struct {
	WaitToleranceDays map[string]Quantile `json:"waitToleranceDays"` // by priority
	Cancellation      Cancellation        `json:"cancellation"`
	NoShow            NoShow              `json:"noShow"`
	Arrival           []ArrivalClass      `json:"arrivalClasses"`
	WalkInPatienceMin Quantile            `json:"walkInPatienceMin"`
	Registration      Registration        `json:"registration"`
}

// Registration describes check-in time at the desk.
type Registration struct {
	ScheduledSec    Quantile `json:"scheduledSec"`
	WalkInSec       Quantile `json:"walkInSec"`
	OpsFactorBP     int64    `json:"opsFactorBP"`     // operations manager registering
	SelfRegFactorBP int64    `json:"selfRegFactorBP"` // technologist registering
}

// NoShow describes missed appointments: a base rate per modality plus a
// lead-time effect, and the chance a no-show asks for a new date.
type NoShow struct {
	BasePPM           map[string]int64 `json:"basePPM"`
	PerTenDaysLeadPPM int64            `json:"perTenDaysLeadPPM"`
	MaxLeadPPM        int64            `json:"maxLeadPPM"`
	RebookPPM         int64            `json:"rebookPPM"`
}

// ArrivalClass is a punctuality class: offsets from the appointment start
// in [fromMin, toMin] minutes (negative is early).
type ArrivalClass struct {
	Name     string `json:"name"`
	WeightBP int64  `json:"weightBP"`
	FromMin  int64  `json:"fromMin"`
	ToMin    int64  `json:"toMin"`
}

// WalkIns describes patients who arrive with an order but no appointment.
type WalkIns struct {
	DailyMilli      int64            `json:"dailyMilli"`
	HourProfileBP   map[string]int64 `json:"hourProfileBP"`
	WeekdayFactorBP map[string]int64 `json:"weekdayFactorBP"`
	PriorityMixBP   map[string]int64 `json:"priorityMixBP"`
	ExamMix         map[string]int64 `json:"examMix"`
}

// Cancellation describes appointment cancellations.
type Cancellation struct {
	PPM       int64    `json:"ppm"`
	LeadHours Quantile `json:"leadHours"`
	RebookPPM int64    `json:"rebookPPM"`
}

// OpeningAppt is one appointment on a hand-built opening schedule.
type OpeningAppt struct {
	Day      int64  `json:"day,omitempty"` // days after startDate
	Time     string `json:"time"`
	Room     string `json:"room"`
	Exam     string `json:"exam"`
	Priority string `json:"priority,omitempty"` // default routine
	Source   string `json:"source"`
}

// Room is one room (one scanner in V0.1).
type Room struct {
	ID       string `json:"id"`
	Modality string `json:"modality"`
	Name     string `json:"name"`
}

// Employee is one staff member and their weekly roster.
type Employee struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Role             string            `json:"role"`
	Skills           []string          `json:"skills"`
	Pay              Pay               `json:"pay"`
	SiteAllocationBP int64             `json:"siteAllocationBP,omitempty"`
	Roster           map[string]*Shift `json:"roster"`
}

// Pay is hourly (rateCents per hour) or salaried (rateCents per year).
type Pay struct {
	Type      string `json:"type"`
	RateCents int64  `json:"rateCents"`
}

// Shift is a roster entry for one weekday. Weeks restricts it to rota week
// "A" or "B" (fortnight rota); empty means every week.
type Shift struct {
	Start      string      `json:"start"`
	End        string      `json:"end"`
	Weeks      string      `json:"weeks,omitempty"`
	Breaks     [][2]string `json:"breaks,omitempty"`
	Assignment string      `json:"assignment"`
	Covers     []Cover     `json:"covers,omitempty"`
	Location   string      `json:"location,omitempty"` // "onsite" (default) or "remote"
}

// Cover is a window in which a technologist also staffs another room. From
// and Until default to the shift's start and end.
type Cover struct {
	Room  string `json:"room"`
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
}

// Policies are the operating rules.
type Policies struct {
	WalkIn               string `json:"walkIn"`
	Worklist             string `json:"worklist"`
	Late                 string `json:"late"`
	GraceMin             int64  `json:"graceMin"`
	OvertimeAllowanceMin int64  `json:"overtimeAllowanceMin"`
	MaxShiftHours        int64  `json:"maxShiftHours"`
	ContrastSupervision  string `json:"contrastSupervision"`
	Booking              string `json:"booking"`
	BookingHorizonDays   int64  `json:"bookingHorizonDays"`
	BookingNoticeMin     int64  `json:"bookingNoticeMin"`
	EarlyStartMin        int64  `json:"earlyStartMin"`
	StartEarly           bool   `json:"startEarly"`
	BreakDeferMaxMin     int64  `json:"breakDeferMaxMin"`
	WeeklyOvertimeHours  int64  `json:"weeklyOvertimeHours"`
	OvertimeFactorBP     int64  `json:"overtimeFactorBP"`
}

// maxExtendsDepth bounds extends chains.
const maxExtendsDepth = 8

// Load reads the scenario at name from fsys, resolving extends (paths are
// relative to the root of fsys) and the preset overlay
// (presets/<preset>.json), then decodes it strictly.
func Load(fsys fs.FS, name string) (*File, error) {
	m, err := loadChain(fsys, name, nil)
	if err != nil {
		return nil, err
	}
	if p, ok := m["preset"].(string); ok && p != "" {
		pm, err := readObject(fsys, path.Join("presets", p+".json"))
		if err != nil {
			return nil, fmt.Errorf("preset %q: %w", p, err)
		}
		if _, ok := pm["extends"]; ok {
			return nil, fmt.Errorf("preset %q: presets cannot use extends", p)
		}
		delete(pm, "preset")
		m = Merge(m, pm).(map[string]any)
	}
	delete(m, "extends")
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f File
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &f, nil
}

func loadChain(fsys fs.FS, name string, seen []string) (map[string]any, error) {
	for _, s := range seen {
		if s == name {
			return nil, fmt.Errorf("extends cycle: %s → %s", strings.Join(seen, " → "), name)
		}
	}
	if len(seen) >= maxExtendsDepth {
		return nil, fmt.Errorf("extends chain deeper than %d at %s", maxExtendsDepth, name)
	}
	m, err := readObject(fsys, name)
	if err != nil {
		return nil, err
	}
	base, ok := m["extends"]
	if !ok {
		return m, nil
	}
	bs, ok := base.(string)
	if !ok || bs == "" {
		return nil, fmt.Errorf("%s: extends must be a file path", name)
	}
	bm, err := loadChain(fsys, bs, append(seen, name))
	if err != nil {
		return nil, err
	}
	delete(m, "extends")
	return Merge(bm, m).(map[string]any), nil
}

func readObject(fsys fs.FS, name string) (map[string]any, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if m == nil {
		return nil, errors.New(name + ": not a JSON object")
	}
	return m, nil
}

// Merge overlays o onto b by key path and returns the result:
//
//   - objects merge recursively; a null value in o deletes the key;
//   - arrays whose elements (in both b and o) are all objects with a string
//     "id" merge by id: matching elements merge recursively, new ones are
//     appended in o's order;
//   - anything else in o replaces b.
//
// Neither input is modified.
func Merge(b, o any) any {
	switch ov := o.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return clean(ov)
		}
		out := make(map[string]any, len(bv)+len(ov))
		for k, v := range bv {
			out[k] = v
		}
		for k, v := range ov {
			if v == nil {
				delete(out, k)
				continue
			}
			if cur, ok := out[k]; ok {
				out[k] = Merge(cur, v)
			} else {
				out[k] = clean(v)
			}
		}
		return out
	case []any:
		bv, ok := b.([]any)
		if !ok || !allHaveIDs(bv) || !allHaveIDs(ov) {
			return clean(ov)
		}
		out := append([]any(nil), bv...)
		for _, e := range ov {
			id := e.(map[string]any)["id"]
			found := false
			for i, x := range out {
				if x.(map[string]any)["id"] == id {
					out[i] = Merge(x, e)
					found = true
					break
				}
			}
			if !found {
				out = append(out, clean(e))
			}
		}
		return out
	}
	return o
}

// clean drops null-valued keys from objects so that a null in an overlay
// means "absent" even where there was nothing to delete.
func clean(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return Merge(map[string]any{}, x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = clean(e)
		}
		return out
	}
	return v
}

func allHaveIDs(xs []any) bool {
	if len(xs) == 0 {
		return false
	}
	for _, x := range xs {
		m, ok := x.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := m["id"].(string); !ok {
			return false
		}
	}
	return true
}
