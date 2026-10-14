package rng

import (
	"fmt"
	"math/bits"
)

// Fixed point: probabilities are stored as round(p * 2^62).
const (
	fracBits = 62
	fpOne    = uint64(1) << fracBits
	// tailEps is the truncation threshold 2^-40 expressed in fixed point.
	tailEps = uint64(1) << (fracBits - 40)
	// maxPartMilli bounds the rate of one table. Larger rates are split
	// into equal parts so that e^-λ keeps enough fixed-point precision.
	maxPartMilli = 16_000
	// MaxPoissonMilli is the largest supported rate (λ = 200).
	MaxPoissonMilli = 200_000
)

// Poisson samples Poisson-distributed counts for one fixed rate, using an
// integer-only cumulative table and inverse transform.
//
// Rates above 16 are split into n = ceil(λ/16) equal parts; a sample is the
// sum of n part samples (exactly Poisson, since sums of independent Poisson
// variables are Poisson). Each part consumes one Uniform draw.
type Poisson struct {
	milli uint32
	parts int
	cdf   []uint64 // cumulative, fixed point; last entry == fpOne
}

// NewPoisson builds a sampler for λ = lambdaMilli/1000.
func NewPoisson(lambdaMilli uint32) (*Poisson, error) {
	if lambdaMilli > MaxPoissonMilli {
		return nil, fmt.Errorf("rng: Poisson rate %d.%03d exceeds 200", lambdaMilli/1000, lambdaMilli%1000)
	}
	parts := 1
	if lambdaMilli > maxPartMilli {
		parts = int((lambdaMilli + maxPartMilli - 1) / maxPartMilli)
	}
	// Each part has rate lambdaMilli/parts. Keep it exact by working with
	// numerator lambdaMilli and denominator 1000*parts.
	den := uint64(1000 * parts)
	num := uint64(lambdaMilli)

	p := expNeg(num, den)
	var cdf []uint64
	cum := uint64(0)
	for k := uint64(1); ; k++ {
		cum += p
		if cum >= fpOne || fpOne-cum < tailEps {
			cdf = append(cdf, fpOne)
			break
		}
		cdf = append(cdf, cum)
		// p_k = p_{k-1} * λ / k  with λ = num/den.
		hi, lo := bits.Mul64(p, num)
		p, _ = bits.Div64(hi, lo, den*k)
		if p == 0 {
			cdf[len(cdf)-1] = fpOne
			break
		}
	}
	return &Poisson{milli: lambdaMilli, parts: parts, cdf: cdf}, nil
}

// LambdaMilli returns the rate in thousandths.
func (p *Poisson) LambdaMilli() uint32 { return p.milli }

// Parts returns how many draws one Sample consumes.
func (p *Poisson) Parts() int { return p.parts }

// Sample draws one count (Parts draws).
func (p *Poisson) Sample(src Source) int {
	n := 0
	for i := 0; i < p.parts; i++ {
		u := Uniform(src, fpOne)
		lo, hi := 0, len(p.cdf)-1
		for lo < hi {
			mid := (lo + hi) / 2
			if u < p.cdf[mid] {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		n += lo
	}
	return n
}

// expNeg returns e^(-num/den) in fixed point, integer-only:
// x = (num/den)/2^r reduced below 1/16, Taylor series for e^-x, then r
// squarings. Requires num/den ≤ 16.
func expNeg(num, den uint64) uint64 {
	r := uint(0)
	for num*16 >= den<<r { // until x = num/(den*2^r) < 1/16
		r++
	}
	// x in fixed point = num * 2^(62-r) / den, computed in 128 bits.
	hi, lo := shl128(num, fracBits-r)
	x, _ := bits.Div64(hi, lo, den)

	// Taylor series of e^-x with signed terms: sum (-x)^n / n!.
	sum := int64(fpOne)
	term := fpOne
	for n := uint64(1); n <= 20; n++ {
		term = fpMul(term, x) / n
		if term == 0 {
			break
		}
		if n%2 == 1 {
			sum -= int64(term)
		} else {
			sum += int64(term)
		}
	}
	y := uint64(sum)
	for i := uint(0); i < r; i++ {
		y = fpMul(y, y)
	}
	return y
}

// fpMul multiplies two fixed-point values (each < 2^63), truncating.
func fpMul(a, b uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	return hi<<(64-fracBits) | lo>>fracBits
}

// shl128 returns v << s as a 128-bit (hi, lo) pair, for s < 64.
func shl128(v uint64, s uint) (hi, lo uint64) {
	if s == 0 {
		return 0, v
	}
	return v >> (64 - s), v << s
}
