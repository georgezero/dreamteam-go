package rng

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
)

func TestKeyBytesEncoding(t *testing.T) {
	var seed Seed
	for i := range seed {
		seed[i] = byte(i)
	}
	got := KeyBytes(seed, "exam.roomTime", 7, -1)
	var want []byte
	want = append(want, "dreamteam/v1"...)
	want = append(want, seed[:]...)
	want = append(want, 13) // uvarint(len("exam.roomTime"))
	want = append(want, "exam.roomTime"...)
	want = append(want, 0x01, 0, 0, 0, 0, 0, 0, 0, 7)
	want = append(want, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)
	if !bytes.Equal(got, want) {
		t.Fatalf("KeyBytes\n got %x\nwant %x", got, want)
	}
}

// Golden values pin the derivation and generator. Changing them changes
// every simulation outcome, so update only on purpose.
func TestGolden(t *testing.T) {
	seed := SeedFromInt(1)
	if got := hex.EncodeToString(seed[:8]); got != goldenSeedPrefix {
		t.Errorf("SeedFromInt(1) prefix = %s", got)
	}
	h := sha256.Sum256(KeyBytes(seed, "golden", 1, 2))
	if got := hex.EncodeToString(h[:8]); got != goldenKeyPrefix {
		t.Errorf("stream key prefix = %s", got)
	}
	s := New(seed, "golden", 1, 2)
	for i, want := range goldenUint64 {
		if got := s.Uint64(); got != want {
			t.Errorf("draw %d = %#x, want %#x", i, got, want)
		}
	}
}

// Seed and key prefixes were cross-checked with an independent Python
// implementation of the derivation.
const (
	goldenSeedPrefix = "2e15774a3e87e79f"
	goldenKeyPrefix  = "86573f9320847a07"
)

var goldenUint64 = []uint64{0xee4a3fc63936af54, 0xb16b769c85d35b56, 0x964ce46df1523769}

func TestStreamsIndependentAndRepeatable(t *testing.T) {
	seed := SeedFromInt(42)
	a1 := New(seed, "a", 1).Uint64()
	a2 := New(seed, "a", 1).Uint64()
	b := New(seed, "b", 1).Uint64()
	a3 := New(seed, "a", 2).Uint64()
	c := New(SeedFromInt(43), "a", 1).Uint64()
	if a1 != a2 {
		t.Fatal("same key gave different output")
	}
	if a1 == b || a1 == a3 || a1 == c {
		t.Fatal("different keys collided")
	}
	// Domain/key boundary must not be ambiguous.
	if bytes.Equal(KeyBytes(seed, "ab"), KeyBytes(seed, "a", 0x62)) {
		t.Fatal("ambiguous encoding")
	}
}

func TestUniformBernoulliWeighted(t *testing.T) {
	s := New(SeedFromInt(7), "test")
	const n = 200_000
	hits := 0
	var counts [3]int
	for i := 0; i < n; i++ {
		if v := s.Uniform(10); v >= 10 {
			t.Fatalf("Uniform out of range: %d", v)
		}
		if s.Bernoulli(250_000) {
			hits++
		}
		counts[s.Weighted([]uint64{1, 0, 3})]++
	}
	if rate := float64(hits) / n; math.Abs(rate-0.25) > 0.005 {
		t.Errorf("Bernoulli rate %.4f", rate)
	}
	if counts[1] != 0 {
		t.Error("zero weight chosen")
	}
	if r := float64(counts[2]) / n; math.Abs(r-0.75) > 0.005 {
		t.Errorf("Weighted share %.4f", r)
	}
	if s.Bernoulli(0) || !s.Bernoulli(OneMillion) {
		t.Error("Bernoulli extremes")
	}
}

func TestQuantile(t *testing.T) {
	bad := [][]QuantilePoint{
		{{0, 1}},
		{{1, 1}, {OneMillion, 2}},
		{{0, 1}, {500_000, 3}, {500_000, 4}, {OneMillion, 5}},
		{{0, 5}, {OneMillion, 4}},
	}
	for i, pts := range bad {
		if _, err := NewQuantile(pts); err == nil {
			t.Errorf("bad table %d accepted", i)
		}
	}
	q, err := NewQuantile([]QuantilePoint{{0, 360}, {100_000, 480}, {500_000, 600}, {900_000, 900}, {OneMillion, 1320}})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[PPM]int64{0: 360, 50_000: 420, 100_000: 480, 500_000: 600, 700_000: 750, 999_999: 1319}
	for u, want := range cases {
		if got := q.At(u); got != want {
			t.Errorf("At(%d) = %d, want %d", u, got, want)
		}
	}
	s := New(SeedFromInt(3), "q")
	below := 0
	for i := 0; i < 100_000; i++ {
		v := q.Sample(s)
		if v < q.Min() || v > q.Max() {
			t.Fatalf("sample %d outside [%d,%d]", v, q.Min(), q.Max())
		}
		if v < 600 {
			below++
		}
	}
	if r := float64(below) / 100_000; math.Abs(r-0.5) > 0.01 {
		t.Errorf("median share %.3f", r)
	}
}

func TestPoissonTable(t *testing.T) {
	for _, milli := range []uint32{0, 250, 1000, 3500, 16_000} {
		p, err := NewPoisson(milli)
		if err != nil {
			t.Fatal(err)
		}
		if p.cdf[len(p.cdf)-1] != fpOne {
			t.Fatalf("λ=%d: cdf does not end at 1", milli)
		}
		lambda := float64(milli) / 1000
		p0 := float64(p.cdf[0]) / float64(fpOne)
		if want := math.Exp(-lambda); math.Abs(p0-want) > 1e-12*math.Max(want, 1e-6) && math.Abs(p0-want) > 1e-15 {
			t.Errorf("λ=%v: P(0) = %.17g, want %.17g", lambda, p0, want)
		}
		for k := 1; k < len(p.cdf); k++ {
			if p.cdf[k] < p.cdf[k-1] {
				t.Fatalf("λ=%v: cdf decreasing at %d", lambda, k)
			}
		}
	}
	if _, err := NewPoisson(MaxPoissonMilli + 1); err == nil {
		t.Error("accepted λ > 200")
	}
}

func TestPoissonMoments(t *testing.T) {
	for _, milli := range []uint32{500, 3000, 40_000} { // 40 uses split parts
		p, err := NewPoisson(milli)
		if err != nil {
			t.Fatal(err)
		}
		s := New(SeedFromInt(9), "poisson", int64(milli))
		const n = 100_000
		var sum, sumSq float64
		for i := 0; i < n; i++ {
			v := float64(p.Sample(s))
			sum += v
			sumSq += v * v
		}
		lambda := float64(milli) / 1000
		mean := sum / n
		variance := sumSq/n - mean*mean
		if math.Abs(mean-lambda) > 4*math.Sqrt(lambda/n) {
			t.Errorf("λ=%v: mean %.4f", lambda, mean)
		}
		if math.Abs(variance-lambda)/lambda > 0.03 {
			t.Errorf("λ=%v: variance %.4f", lambda, variance)
		}
	}
}

// BenchmarkKeyedDraw measures one keyed draw (SHA-256 key derivation,
// ChaCha8 setup, one sample) — the per-attribute cost in the simulation.
func BenchmarkKeyedDraw(b *testing.B) {
	seed := SeedFromInt(1)
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink += New(seed, "exam.roomTime", int64(i), 3, 0).Uniform(1_000_000)
	}
	_ = sink
}

func TestHashDraws(t *testing.T) {
	seed := SeedFromInt(1)
	d := Hash(seed, "golden", 1, 2)
	sum := sha256.Sum256(KeyBytes(seed, "golden", 1, 2))
	if got := d.Uint64(); hex.EncodeToString(sum[:8]) != goldenKeyPrefix || got != 0x86573f9320847a07 {
		t.Fatalf("first lane %#x", got)
	}
	d.Uint64()
	d.Uint64()
	d.Uint64()
	defer func() {
		if recover() == nil {
			t.Fatal("fifth draw did not panic")
		}
	}()
	d.Uint64()
}

// BenchmarkHashDraw measures a single-key draw without a stream.
func BenchmarkHashDraw(b *testing.B) {
	seed := SeedFromInt(1)
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink += Hash(seed, "exam.roomTime", int64(i), 3, 0).Uniform(1_000_000)
	}
	_ = sink
}
