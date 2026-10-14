package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// eventLog keeps the observable log: a bounded ring of recent entries plus a
// running hash over every entry ever emitted, so the digest does not depend
// on retention (SIMULATION_DESIGN §4.5).
//
// The hash is one SHA-256 stream over the entries' canonical encoding
// (sequence, time, severity, category, text), finalized only when asked
// for. Entries keep their time as a number; the ISO text is made when the
// log is observed.
type eventLog struct {
	ring  []logEntry
	start int // index of the oldest entry in ring
	total uint64
	h     hash.Hash
	buf   []byte
}

type logEntry struct {
	seq  uint64
	at   simtime.Time
	sev  api.Severity
	cat  string
	text string
}

const logRetention = 2000

func (l *eventLog) emit(at simtime.Time, sev api.Severity, cat, text string) {
	l.total++
	if l.h == nil {
		l.h = sha256.New()
	}
	b := l.buf[:0]
	b = binary.BigEndian.AppendUint64(b, l.total)
	b = binary.BigEndian.AppendUint64(b, uint64(at))
	for _, s := range []string{string(sev), cat, text} {
		b = binary.AppendUvarint(b, uint64(len(s)))
		b = append(b, s...)
	}
	l.h.Write(b)
	l.buf = b

	e := logEntry{seq: l.total, at: at, sev: sev, cat: cat, text: text}
	if len(l.ring) < logRetention {
		l.ring = append(l.ring, e)
		return
	}
	l.ring[l.start] = e
	l.start = (l.start + 1) % logRetention
}

// digest returns the running hash of every entry so far.
func (l *eventLog) digest() []byte {
	if l.h == nil {
		l.h = sha256.New()
	}
	return l.h.Sum(nil)
}

// recent returns up to n most recent entries, oldest first.
func (l *eventLog) recent(cal simtime.Calendar, n int) []api.LogEntry {
	if n > len(l.ring) {
		n = len(l.ring)
	}
	out := make([]api.LogEntry, 0, n)
	for i := len(l.ring) - n; i < len(l.ring); i++ {
		e := &l.ring[(l.start+i)%len(l.ring)]
		out = append(out, api.LogEntry{Seq: e.seq, At: e.at, Time: cal.ISO(e.at), Severity: e.sev, Category: e.cat, Text: e.text})
	}
	return out
}
