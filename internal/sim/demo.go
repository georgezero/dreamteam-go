package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Demo source for V0.1 step 4: synthetic visits that give the TUI shell
// something to show. It is scaffolding and will be removed once the real
// patient flow exists (steps 6–8).

var demoHours = map[simtime.Weekday][2]simtime.Duration{
	simtime.Monday:    {8 * simtime.Hour, 20 * simtime.Hour},
	simtime.Tuesday:   {8 * simtime.Hour, 20 * simtime.Hour},
	simtime.Wednesday: {8 * simtime.Hour, 20 * simtime.Hour},
	simtime.Thursday:  {8 * simtime.Hour, 20 * simtime.Hour},
	simtime.Friday:    {8 * simtime.Hour, 20 * simtime.Hour},
	simtime.Saturday:  {10 * simtime.Hour, 18 * simtime.Hour},
}

var demoModalities = []string{"XR", "US", "CT", "MR"}

type demoState struct {
	visits int64
	inside int64
	rate   *rng.Poisson
}

// demoSchedule draws each open hour's arrivals (about 6 per hour).
func (st *state) demoSchedule(e *Engine, day int64, open, closeAt simtime.Time) {
	if st.demo.rate == nil {
		p, err := rng.NewPoisson(6000)
		if err != nil {
			panic(err)
		}
		st.demo.rate = p
	}
	for block := int64(0); open.Add(simtime.Duration(block)*simtime.Hour) < closeAt; block++ {
		blockStart := open.Add(simtime.Duration(block) * simtime.Hour)
		n := st.demo.rate.Sample(rng.Hash(st.seed, "demo.count", day, block))
		for k := 0; k < n; k++ {
			d := rng.Hash(st.seed, "demo.visit", day, block, int64(k))
			at := blockStart.Add(simtime.Duration(d.Uniform(uint64(simtime.Hour))))
			mod := int64(d.Uniform(uint64(len(demoModalities))))
			stay := simtime.Duration(10*60 + d.Uniform(30*60))
			e.Push(Event{At: at, Class: ClassArrival, Kind: kindDemoArrival, A: mod, B: int64(stay)})
		}
	}
}

func (st *state) demoHandle(e *Engine, ev Event) {
	mod := demoModalities[ev.A]
	switch ev.Kind {
	case kindDemoArrival:
		st.demo.visits++
		st.demo.inside++
		id := st.demo.visits
		st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("Demo patient D%04d arrived for %s", id, mod))
		e.Push(Event{At: e.Now().Add(simtime.Duration(ev.B)), Class: ClassCompletion, Kind: kindDemoDone, A: ev.A, B: id})
	case kindDemoDone:
		st.demo.inside--
		st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("Demo patient D%04d finished %s", ev.B, mod))
	}
}
