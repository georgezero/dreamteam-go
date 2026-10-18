package eval

import (
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// ocQueue is a synthetic reading queue for the backlog operating
// characteristics (Slices 5C.2 and 5C.3): the scenario's demand shape and
// reading hours, with the departures from a stable queue each scenario
// needs. The zero value of every departure is the Slice 5C.2 queue, which
// draws the same random stream.
type ocQueue struct {
	Seasonal  bool    `json:"seasonal"`
	Util      float64 `json:"utilization"`          // of the season-1 demand (0.85 when 0)
	DayCV     float64 `json:"dailyArrivalCV"`       // gamma day factor on arrivals (bursty days)
	Capacity  []ocCap `json:"capacity,omitempty"`   // reading capacity factors by day
	Inherit   int     `json:"inheritedParked"`      // routine studies acquired before the takeover, unread
	InheritTo int     `json:"inheritedParkedToDay"` // until this day
	// Park: from ParkFrom to ParkTo, the first routine arrival of each
	// weekday is held back ParkDays before it joins the queue (old work
	// whose count does not grow).
	ParkFrom int `json:"parkFromDay,omitempty"`
	ParkTo   int `json:"parkToDay,omitempty"`
	ParkDays int `json:"parkDays,omitempty"`
	// Erosion: from ErodeFrom, capacity falls by ErodePerDay of the
	// season-1 capacity each day (a slow creep).
	ErodeFrom   int     `json:"erodeFromDay,omitempty"`
	ErodePerDay float64 `json:"erodePerDay,omitempty"`
}

// ocCap multiplies reading capacity by Factor on days [From, To) (To 0:
// to the end).
type ocCap struct {
	From   int     `json:"fromDay"`
	To     int     `json:"toDay,omitempty"`
	Factor float64 `json:"factor"`
}

func (q ocQueue) capacity(day int) float64 {
	f := 1.0
	if q.ErodePerDay > 0 && day >= q.ErodeFrom {
		f = max(0, 1-q.ErodePerDay*float64(day-q.ErodeFrom))
	}
	for _, c := range q.Capacity {
		if day >= c.From && (c.To == 0 || day < c.To) {
			f *= c.Factor
		}
	}
	return f
}

// record simulates the queue hour by hour from the takeover (Tue
// 2026-10-13 06:00) for 181 days: arrivals at 52 a day (Poisson, the
// weekday profile, the hourly profile 08:00–20:00, the season factor when
// seasonal, the day factor when bursty), reading capacity Mon–Fri
// 07:30–20:00 and Sat 10:00–14:00 at the utilization of the season-1
// demand times the capacity factor; urgent first, then soon, then
// routine, first in first out (held-back studies join at the front). It
// returns a record with the studies and the daily 06:00 samples.
func (q ocQueue) record(r ocRNG) *Record {
	const perDay = 52.0
	util := q.Util
	if util == 0 {
		util = 0.85
	}
	start := t0 // 06:00 on the takeover date
	date, _ := simtime.ParseDate(t0Date)
	rec := &Record{TakeoverSec: start, TakeoverDate: t0Date, HorizonEndSec: start + 150*daySec, TATTargetMin: testTargets}
	// Weekly reading hours: 5 × 12.5 + 4 = 66.5; capacity per reading hour.
	capPerHour := perDay * 7 / 66.5 / util
	var queue [3][]int // indices into rec.Studies, by priority (0 urgent)
	type held struct {
		i     int
		until int64
	}
	var parked []held
	for k := range q.Inherit {
		acq := start - 3*daySec + int64(k)*60
		rec.Studies = append(rec.Studies, Study{ID: "inh", Priority: "routine", AcquiredSec: acq, EnteredSec: acq})
		parked = append(parked, held{len(rec.Studies) - 1, start + int64(q.InheritTo)*daySec})
	}
	carry, dayFactor := 0.0, 1.0
	parkedDay := -1
	sample := func(d int, at int64) {
		s := Sample{Day: d, AtSec: at, Date: date.AddDays(int64(d)).String()}
		for k := range queue {
			s.Unread += len(queue[k])
		}
		s.Unread += len(parked)
		for i := range rec.Studies {
			if rec.Studies[i].AcquiredSec <= at && rec.Studies[i].AcquiredSec > start {
				s.AcquiredCum++
			}
			if f := rec.Studies[i].FinalSec; f != 0 && f <= at {
				s.FinalCum++
			}
		}
		rec.Samples = append(rec.Samples, s)
	}
	for h := 0; h < 181*24; h++ {
		at := start + int64(h)*hour
		clock := (6 + h) % 24
		day := (6 + h) / 24 // days after the takeover date's midnight
		dt := date.AddDays(int64(day))
		wd := (int(dt.Weekday()) + 6) % 7 // Monday 0
		if h > 0 && clock == 6 {
			sample(day, at)
		}
		if clock == 0 && q.DayCV > 0 {
			k := 1 / (q.DayCV * q.DayCV)
			dayFactor = r.gamma(k) / k
		}
		// Held-back studies due back join the front of their queue.
		for i := 0; i < len(parked); {
			if parked[i].until <= at {
				queue[2] = append([]int{parked[i].i}, queue[2]...)
				parked = append(parked[:i], parked[i+1:]...)
				continue
			}
			i++
		}
		// Arrivals this hour.
		if clock >= 8 && clock < 20 {
			lam := perDay * ocWeekday[wd] / 12 * dayFactor
			if q.Seasonal {
				lam *= ocSeason[dt.Month-1]
			}
			for range r.poisson(lam) {
				u := r.Float64()
				k := 2
				switch {
				case u < 0.02:
					k = 0
				case u < 0.14:
					k = 1
				}
				acq := at + int64(r.Float64()*3600)
				rec.Studies = append(rec.Studies, Study{ID: "s", Priority: Priorities[k], AcquiredSec: acq, EnteredSec: acq})
				if k == 2 && q.ParkDays > 0 && day >= q.ParkFrom && day < q.ParkTo && wd < 5 && parkedDay != day {
					parkedDay = day
					parked = append(parked, held{len(rec.Studies) - 1, acq + int64(q.ParkDays)*daySec})
					continue
				}
				queue[k] = append(queue[k], len(rec.Studies)-1)
			}
		}
		// Reading this hour.
		open := (wd < 5 && clock >= 8 && clock < 20) || (wd < 5 && clock == 7) || (wd == 5 && clock >= 10 && clock < 14)
		if open {
			c := capPerHour * q.capacity(day)
			if wd < 5 && clock == 7 {
				c /= 2 // from 07:30
			}
			carry += c
			for carry >= 1 {
				fin := at + hour - int64(carry/c*3600)
				k := 0
				for k < 3 && (len(queue[k]) == 0 || rec.Studies[queue[k][0]].AcquiredSec >= fin) {
					k++
				}
				if k == 3 {
					carry = min(carry, 1) // nothing ready: idle, no capacity banked
					break
				}
				s := &rec.Studies[queue[k][0]]
				queue[k] = queue[k][1:]
				st := max(fin-60, s.EnteredSec, start)
				s.ClaimedSec, s.ReadStartSec, s.FinalSec = st, st, max(fin, st)
				carry--
			}
		}
	}
	rec.ObservedUntilSec = start + 181*daySec
	return rec
}

// ocQueueRecord is the Slice 5C.2 queue (oc-v0.2-eval-3.json): stationary
// or seasonal demand, capacity lower by loss from day lossFrom.
func ocQueueRecord(r ocRNG, seasonal bool, lossFrom int, loss float64) *Record {
	q := ocQueue{Seasonal: seasonal}
	if lossFrom > 0 {
		q.Capacity = []ocCap{{From: lossFrom, Factor: 1 - loss}}
	}
	return q.record(r)
}
