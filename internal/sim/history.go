package sim

import (
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Daily history: one summary per day, recorded at accounting close, so
// the practice taken over at day 1 comes with its recent record (demand,
// utilization, waits, turnaround, money). Days before the start date are
// history (shown in trends, never scored); "today" figures never include
// them.

type dayRecord struct {
	day                               int64
	received, booked, lost, cancelled int
	walkIns, arrived, noShows, exams  int
	reportsFinal, unreadAtClose       int
	waitMedian, tatMedian             simtime.Duration
	overtime                          simtime.Duration
	exByMod                           [maxMods]int
	busy, open, staffed               [maxMods]simtime.Duration
}

// recordDay summarizes today at accounting close (after no-shows are
// finalized).
func (st *state) recordDay() {
	ds := st.dem.days[st.day]
	r := dayRecord{day: st.day, unreadAtClose: len(st.dem.worklist)}
	for _, m := range st.mods {
		c := st.dem.today[m]
		r.received += c.received
		r.booked += c.booked
		r.lost += c.lost
		r.cancelled += c.cancelled
	}
	f := &st.dem.flowToday
	r.walkIns, r.arrived, r.noShows, r.exams = f.walkIns, f.arrived+f.walkIns, f.noShows, f.examsDone
	r.reportsFinal, r.overtime = f.reportsFinal, f.otSec
	var waits, tats []int64
	for _, vi := range ds.visits {
		v := &st.dem.visits[vi]
		if v.study < 0 {
			continue
		}
		waits = append(waits, int64(v.examStart.Sub(v.arrived)))
		if k := st.modIndex(st.cfg.Rooms[v.room].Modality); k >= 0 {
			r.exByMod[k]++
			r.busy[k] += v.examEnd.Sub(v.examStart)
		}
	}
	for _, si := range st.dem.finals[st.histFinals:] {
		s := &st.dem.studies[si]
		tats = append(tats, int64(s.finalAt.Sub(s.acquiredAt)))
	}
	st.histFinals = len(st.dem.finals)
	r.waitMedian, r.tatMedian = simtime.Duration(median(waits)), simtime.Duration(median(tats))
	if h := st.plan.Hours; h != nil {
		for ri, rc := range st.cfg.Rooms {
			k := st.modIndex(rc.Modality)
			if k < 0 {
				continue
			}
			r.open[k] += h.To - h.From
			for _, seg := range st.cov.Segments {
				if seg.Open && len(seg.Rooms[ri]) > 0 {
					r.staffed[k] += seg.To - seg.From
				}
			}
		}
	}
	st.history = append(st.history, r)
}

func utilBP(busy, of simtime.Duration) int64 {
	if of <= 0 {
		return 0
	}
	return int64(busy) * 10_000 / int64(of)
}

// History returns the summaries of up to n completed days, oldest first.
func (w *World) History(n int) []api.DaySummary {
	st := w.st
	led := &st.fin.led
	h := st.history[max(0, len(st.history)-n):]
	var out []api.DaySummary
	for _, r := range h {
		d := st.cal.Date(st.cal.StartOfDay(r.day))
		s := api.DaySummary{Date: d.String(), Weekday: d.Weekday().String(), BeforeStart: r.day < st.startDay,
			Orders: r.received, Booked: r.booked, Lost: r.lost, Cancelled: r.cancelled, WalkIns: r.walkIns,
			Arrived: r.arrived, NoShows: r.noShows, Exams: r.exams, ReportsFinal: r.reportsFinal,
			UnreadAtClose: r.unreadAtClose, WaitMedianMin: int64(r.waitMedian / simtime.Minute),
			TATMedianMin: int64(r.tatMedian / simtime.Minute), OvertimeMin: int64(r.overtime / simtime.Minute)}
		for k, m := range st.mods {
			s.ByModality = append(s.ByModality, api.ModalityDay{Modality: m, Exams: r.exByMod[k],
				BusyMin: int64(r.busy[k] / simtime.Minute), OpenMin: int64(r.open[k] / simtime.Minute),
				StaffedMin: int64(r.staffed[k] / simtime.Minute), UtilOpenBP: utilBP(r.busy[k], r.open[k]),
				UtilStaffedBP: utilBP(r.busy[k], r.staffed[k])})
		}
		if r.day < int64(len(led.days)) {
			b := &led.days[r.day]
			s.NetRevenueCents = -b.net[acctRevenue] - b.net[acctVariance]
			for a := range numAccounts {
				if accountInfo[a].kind == kindExpense {
					s.ExpenseCents += b.net[a]
				}
			}
			s.OperatingProfitCents = s.NetRevenueCents - s.ExpenseCents
			s.CollectionsCents, s.DisbursementsCents = b.receipts, b.paidOut
		}
		s.CashEndCents = led.cashAtEnd(r.day)
		if st.fin.opened && st.cal.StartOfDay(r.day+1) <= st.fin.opening.at {
			s.CashEndCents += st.fin.opening.adjustment // restated, as in the cash history
		}
		out = append(out, s)
	}
	return out
}
