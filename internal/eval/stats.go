package eval

import (
	"maps"
	"math"
	"slices"
)

// Ratio is a count over a denominator. An empty denominator is undefined:
// it is never a zero rate, so it can never pass a gate (policy §1).
type Ratio struct {
	Num     int64   `json:"num"`
	Den     int64   `json:"den"`
	Value   float64 `json:"value"`
	Defined bool    `json:"defined"`
}

func ratio(num, den int64) Ratio {
	if den <= 0 {
		return Ratio{Num: num, Den: den}
	}
	return Ratio{Num: num, Den: den, Value: float64(num) / float64(den), Defined: true}
}

// Dist is a nearest-rank summary of integer observations. N = 0 leaves
// the percentiles zero and Defined false.
type Dist struct {
	N       int   `json:"n"`
	P50     int64 `json:"p50"`
	P90     int64 `json:"p90"`
	Max     int64 `json:"max"`
	Defined bool  `json:"defined"`
}

func dist(xs []int64) Dist {
	if len(xs) == 0 {
		return Dist{}
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	return Dist{N: len(s), P50: rank(s, 50), P90: rank(s, 90), Max: s[len(s)-1], Defined: true}
}

// rank is the nearest-rank percentile of sorted s: the smallest value with
// at least p% of the observations at or below it.
func rank(s []int64, p int) int64 {
	k := (p*len(s) + 99) / 100
	return s[max(k, 1)-1]
}

// Interval is a mean with a two-sided 95% interval. N < 2 leaves the
// interval undefined (a single seed has no spread to estimate).
type Interval struct {
	N       int     `json:"n"`
	Mean    float64 `json:"mean"`
	Lo      float64 `json:"lo"`
	Hi      float64 `json:"hi"`
	Defined bool    `json:"defined"`
}

// pairedT is the mean of paired differences with its Student t 95%
// interval (df = n − 1).
func pairedT(xs []float64) Interval {
	n := len(xs)
	if n == 0 {
		return Interval{}
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	m := sum / float64(n)
	if n < 2 {
		return Interval{N: n, Mean: m, Lo: m, Hi: m}
	}
	var ss float64
	for _, x := range xs {
		ss += (x - m) * (x - m)
	}
	h := tQuantile975(n-1) * math.Sqrt(ss/float64(n-1)/float64(n))
	return Interval{N: n, Mean: m, Lo: m - h, Hi: m + h, Defined: true}
}

// wilson is the Wilson score 95% interval for k of n.
func wilson(k, n int) Interval {
	if n == 0 {
		return Interval{}
	}
	const z = 1.959963984540054
	p := float64(k) / float64(n)
	nn := float64(n)
	c := (p + z*z/(2*nn)) / (1 + z*z/nn)
	h := z / (1 + z*z/nn) * math.Sqrt(p*(1-p)/nn+z*z/(4*nn*nn))
	return Interval{N: n, Mean: p, Lo: max(0, c-h), Hi: min(1, c+h), Defined: true}
}

// tQuantile975 is the 0.975 quantile of Student's t with df degrees of
// freedom, by bisection on the CDF (deterministic, no tables).
func tQuantile975(df int) float64 {
	lo, hi := 0.0, 64.0
	for range 200 {
		mid := (lo + hi) / 2
		if tCDF(mid, float64(df)) < 0.975 {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// tCDF is P(T ≤ t) for t ≥ 0.
func tCDF(t, df float64) float64 {
	return 1 - 0.5*betaInc(df/2, 0.5, df/(df+t*t))
}

// betaInc is the regularized incomplete beta function I_x(a, b).
func betaInc(a, b, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	la, _ := math.Lgamma(a + b)
	lb, _ := math.Lgamma(a)
	lc, _ := math.Lgamma(b)
	front := math.Exp(la - lb - lc + a*math.Log(x) + b*math.Log(1-x))
	if x < (a+1)/(a+b+2) {
		return front * betaCF(a, b, x) / a
	}
	return 1 - front*betaCF(b, a, 1-x)/b
}

// betaCF is the continued fraction for the incomplete beta function
// (modified Lentz).
func betaCF(a, b, x float64) float64 {
	const tiny = 1e-300
	c, d := 1.0, 1-(a+b)*x/(a+1)
	if math.Abs(d) < tiny {
		d = tiny
	}
	d = 1 / d
	h := d
	for m := 1; m <= 300; m++ {
		fm := float64(m)
		aa := fm * (b - fm) * x / ((a + 2*fm - 1) * (a + 2*fm))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		h *= d * c
		aa = -(a + fm) * (a + b + fm) * x / ((a + 2*fm) * (a + 2*fm + 1))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < 1e-15 {
			break
		}
	}
	return h
}

// clusterGroup is one cluster of binary outcomes: its size, its count of
// ones and the value its mean is measured from (0, or the weekday's mean
// rate for a weekday-adjusted day).
type clusterGroup struct {
	n, ones int64
	centre  float64
}

// designEffect is 1 + (m̄ − 1)·ρ, with ρ the one-way ANOVA intra-cluster
// correlation of binary outcomes over the groups and m̄ the mean group
// size: the variance of the overall rate over the binomial variance its
// count implies. 0 when undefined (fewer than two groups, or no
// variation).
func designEffect(gs []clusterGroup) float64 {
	var k, n, ss2 int64
	var sumResid float64
	for _, g := range gs {
		if g.n == 0 {
			continue
		}
		k, n, ss2 = k+1, n+g.n, ss2+g.n*g.n
		sumResid += float64(g.ones) - float64(g.n)*g.centre
	}
	if k < 2 || n <= k {
		return 0
	}
	mean := sumResid / float64(n)
	var ssb, ssw float64
	for _, g := range gs {
		if g.n == 0 {
			continue
		}
		p := float64(g.ones) / float64(g.n)
		d := p - g.centre - mean
		ssb += float64(g.n) * d * d
		ssw += float64(g.n) * p * (1 - p)
	}
	msb, msw := ssb/float64(k-1), ssw/float64(n-k)
	n0 := (float64(n) - float64(ss2)/float64(n)) / float64(k-1)
	if msb+(n0-1)*msw <= 0 {
		return 0
	}
	rho := (msb - msw) / (msb + (n0-1)*msw)
	return 1 + (float64(n)/float64(k)-1)*rho
}

// clusterEffects are a cohort's design effects by practice-local day (the
// weekday's mean rate removed: a fixed weekday pattern adds no variance
// across seeds) and by Monday-based week. Undecided studies are left out.
func clusterEffects(xs []*Study, breach func(*Study) (bool, bool)) (day, week float64) {
	days := map[int64]*clusterGroup{}
	weeks := map[int64]*clusterGroup{}
	var wdN, wdOnes [7]int64
	for _, s := range xs {
		b, decided := breach(s)
		if !decided {
			continue
		}
		ord := s.AcquiredSec / daySec // Time 0 is 00:00 on the epoch: local days are whole multiples
		one := int64(0)
		if b {
			one = 1
		}
		for _, m := range []struct {
			to map[int64]*clusterGroup
			k  int64
		}{{days, ord}, {weeks, floorDiv(ord+3, 7)}} {
			g := m.to[m.k]
			if g == nil {
				g = &clusterGroup{}
				m.to[m.k] = g
			}
			g.n, g.ones = g.n+1, g.ones+one
		}
		wd := (ord%7 + 7) % 7
		wdN[wd], wdOnes[wd] = wdN[wd]+1, wdOnes[wd]+one
	}
	collect := func(m map[int64]*clusterGroup, adj bool) []clusterGroup {
		keys := slices.Sorted(maps.Keys(m))
		out := make([]clusterGroup, 0, len(keys))
		for _, k := range keys {
			g := *m[k]
			if wd := (k%7 + 7) % 7; adj && wdN[wd] > 0 {
				g.centre = float64(wdOnes[wd]) / float64(wdN[wd])
			}
			out = append(out, g)
		}
		return out
	}
	return designEffect(collect(days, true)), designEffect(collect(weeks, false))
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// clusterRobustRate is the pooled rate Σones / Σn over independent
// clusters (seeds) with its t 95% interval from the linearized (ratio
// estimator) variance: no assumption about the dependence inside a
// cluster.
func clusterRobustRate(ones, ns []int64) Interval {
	k := len(ns)
	var so, sn int64
	for i := range ns {
		so, sn = so+ones[i], sn+ns[i]
	}
	if k == 0 || sn == 0 {
		return Interval{}
	}
	r := float64(so) / float64(sn)
	if k < 2 {
		return Interval{N: k, Mean: r, Lo: r, Hi: r}
	}
	var ss float64
	for i := range ns {
		e := float64(ones[i]) - r*float64(ns[i])
		ss += e * e
	}
	se := math.Sqrt(float64(k)/float64(k-1)*ss) / float64(sn)
	h := tQuantile975(k-1) * se
	return Interval{N: k, Mean: r, Lo: max(0, r-h), Hi: min(1, r+h), Defined: true}
}

// binomUpper is the exact (Clopper–Pearson) one-sided upper confidence
// bound, at confidence c, on a proportion with x of n: the u for which
// P(X ≤ x | n, u) = 1 − c (1 when x = n).
func binomUpper(x, n int, c float64) float64 {
	if n == 0 || x >= n {
		return 1
	}
	lo, hi := float64(x)/float64(n), 1.0
	for range 200 {
		mid := (lo + hi) / 2
		// P(X ≤ x) = 1 − I_mid(x+1, n−x)
		if 1-betaInc(float64(x+1), float64(n-x), mid) > 1-c {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}
