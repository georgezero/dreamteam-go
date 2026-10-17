package sim

// Test hooks for teleradiology contracts (Slice 5B).

// TeleFee is a remote claim's locked price.
type TeleFee struct {
	Study          string
	Fee, Surcharge int64
	Night, Final   bool
	Contract       string
}

// TeleFees lists the remote claims' locked prices in claim order.
func (w *World) TeleFees() []TeleFee {
	if w.st.tele == nil {
		return nil
	}
	var out []TeleFee
	for _, cl := range w.st.tele.claims {
		out = append(out, TeleFee{Study: studyID(cl.study), Fee: cl.fee, Surcharge: cl.surcharge, Night: cl.night,
			Final: w.st.dem.studies[cl.study].state == studyFinal, Contract: teleID(cl.contract)})
	}
	return out
}

// TeleTotals are the vendor totals: retainers and read fees recognized,
// and paid.
func (w *World) TeleTotals() (retainer, reads, paid int64) {
	if t := w.st.tele; t != nil {
		return t.retainer, t.reads, t.paid
	}
	return 0, 0, 0
}

// HasTele reports whether the remote reading state exists.
func (w *World) HasTele() bool { return w.st.tele != nil }
