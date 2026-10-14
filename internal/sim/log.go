package sim

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// eventLog keeps the observable log: a bounded ring of recent entries plus a
// running hash over every entry ever emitted, so the digest does not depend
// on retention (SIMULATION_DESIGN §4.5).
type eventLog struct {
	ring  []api.LogEntry
	start int // index of the oldest entry in ring
	total uint64
	hash  [32]byte
}

const logRetention = 2000

func (l *eventLog) emit(cal simtime.Calendar, at simtime.Time, sev api.Severity, cat, text string) {
	l.total++
	e := api.LogEntry{Seq: l.total, At: at, Time: cal.ISO(at), Severity: sev, Category: cat, Text: text}

	var b []byte
	b = append(b, l.hash[:]...)
	b = binary.BigEndian.AppendUint64(b, e.Seq)
	b = binary.BigEndian.AppendUint64(b, uint64(e.At))
	for _, s := range []string{string(e.Severity), e.Category, e.Text} {
		b = binary.AppendUvarint(b, uint64(len(s)))
		b = append(b, s...)
	}
	l.hash = sha256.Sum256(b)

	if len(l.ring) < logRetention {
		l.ring = append(l.ring, e)
		return
	}
	l.ring[l.start] = e
	l.start = (l.start + 1) % logRetention
}

// recent returns up to n most recent entries, oldest first.
func (l *eventLog) recent(n int) []api.LogEntry {
	if n > len(l.ring) {
		n = len(l.ring)
	}
	out := make([]api.LogEntry, 0, n)
	for i := len(l.ring) - n; i < len(l.ring); i++ {
		out = append(out, l.ring[(l.start+i)%len(l.ring)])
	}
	return out
}
