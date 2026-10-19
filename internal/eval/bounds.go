package eval

import (
	"math"
)

// One-sided upper confidence bounds for the mean of independent seed
// values (Slice 5C.4d, docs/calibration/clinical-5c4d/gate-b-statistics.md).

// Bound methods.
const (
	BoundBetting = "betting" // finite-sample, any distribution on [lo, hi]
	BoundT       = "t"       // Student t: approximate, no finite-sample guarantee
)

// upperBound is method's one-sided upper confidence bound, at confidence
// conf, for the mean of xs, values in [lo, hi].
func upperBound(method string, xs []float64, conf, lo, hi float64) float64 {
	switch method {
	case BoundBetting:
		return bettingUpper(xs, conf, lo, hi)
	case BoundT:
		return tUpper(xs, conf)
	}
	return math.Inf(1)
}

// bettingUpper is the betting (capital-process) upper bound: U =
// inf{m : K(m) ≥ 1/α}, K(m) = Π (1 + λᵢ (m − xᵢ)), with each λᵢ chosen
// from x₁ … xᵢ₋₁ only and 0 ≤ λᵢ ≤ c/(hi − lo). K(μ) is then a nonnegative
// martingale under any distribution on [lo, hi] with mean μ, so by
// Ville's inequality P(K(μ) ≥ 1/α) ≤ α; K increases in m (λ does not
// depend on m), so U < μ only when K(μ) ≥ 1/α: P(U < μ) ≤ α at every
// sample size. The λ rule (a predictable plug-in) affects only power.
func bettingUpper(xs []float64, conf, lo, hi float64) float64 {
	n := len(xs)
	if n == 0 {
		return hi
	}
	alpha := 1 - conf
	const c = 0.75
	lmax := c / (hi - lo)
	lambdas := make([]float64, n)
	mean, ss := 0.0, 0.25 // running mean (prior 0) and a prior squared deviation
	for i := range xs {
		v := ss / float64(i+1)
		lambdas[i] = min(math.Sqrt(2*math.Log(1/alpha)/(float64(n)*v)), lmax)
		ss += (xs[i] - mean) * (xs[i] - mean)
		mean += (xs[i] - mean) / float64(i+1)
	}
	logK := func(m float64) float64 {
		s := 0.0
		for i, x := range xs {
			s += math.Log1p(lambdas[i] * (m - x))
		}
		return s
	}
	target := math.Log(1 / alpha)
	if logK(hi) < target {
		return hi
	}
	a, b := lo, hi
	if logK(a) >= target {
		return lo
	}
	for range 200 {
		mid := (a + b) / 2
		if logK(mid) >= target {
			b = mid
		} else {
			a = mid
		}
	}
	return b // the smallest m found to reach 1/α (rounded up)
}

// tUpper is the Student t upper bound x̄ + t(conf, n−1) s / √n.
func tUpper(xs []float64, conf float64) float64 {
	n := len(xs)
	if n < 2 {
		return math.Inf(1)
	}
	mean, sd := meanSD(xs)
	return mean + tQuantile(conf, n-1)*sd/math.Sqrt(float64(n))
}

func meanSD(xs []float64) (mean, sd float64) {
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	if len(xs) < 2 {
		return mean, 0
	}
	ss := 0.0
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(ss / float64(len(xs)-1))
}

// tQuantile is the p quantile (p ≥ 1/2) of Student's t with df degrees
// of freedom, by bisection on tCDF (which holds for t ≥ 0).
func tQuantile(p float64, df int) float64 {
	lo, hi := 0.0, 1e3
	for range 200 {
		mid := (lo + hi) / 2
		if tCDF(mid, float64(df)) < p {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// zeroFloor is method's bound for n seeds that all show no difference:
// the smallest bound the design can give. Above the margin, the design
// cannot establish it whatever the data.
func zeroFloor(method string, n int, conf, lo, hi float64) float64 {
	return upperBound(method, make([]float64, n), conf, lo, hi)
}
