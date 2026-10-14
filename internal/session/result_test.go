package session

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func ninetyDays(b *testing.B) *Session {
	s := openFile(&testing.T{}, "established", 1, nil)
	s.AdvanceTo(s.Now().Add(90 * simtime.Day))
	return s
}

// BenchmarkResult90Days builds and encodes the result after 90 days
// (digests included, replay excluded).
func BenchmarkResult90Days(b *testing.B) {
	s := ninetyDays(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := s.Result(false)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := r.Encode(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompare90Days compares a 90-day result with its own encoding.
func BenchmarkCompare90Days(b *testing.B) {
	s := ninetyDays(b)
	r, _ := s.Result(false)
	raw, _ := r.Encode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if c, err := Compare(raw, r, false); err != nil || !c.Match {
			b.Fatal(err)
		}
	}
}
