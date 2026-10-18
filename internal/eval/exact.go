package eval

import (
	"fmt"
	"math"
	"math/big"
)

// Thresholds are compared exactly: a policy fraction is a whole number of
// basis points (1/10,000), and a rate against it is a comparison of
// integer cross products, so a value exactly at a tolerance is within it
// (no floating-point subtraction decides a gate).

// bp is a policy fraction in basis points; DecodePolicy has checked that
// it is whole.
func bp(f float64) int64 { return int64(math.Round(f * 10_000)) }

func wholeBP(name string, f float64) error {
	if f < 0 || math.Abs(f*10_000-math.Round(f*10_000)) > 1e-6 {
		return fmt.Errorf("%s %v is not a whole number of basis points", name, f)
	}
	return nil
}

func bi(x int64) *big.Int { return big.NewInt(x) }

func mul(xs ...int64) *big.Int {
	r := bi(1)
	for _, x := range xs {
		r.Mul(r, bi(x))
	}
	return r
}

// rateAbove reports num/den > t (basis points), den > 0.
func rateAbove(num, den, t int64) bool { return mul(num, 10_000).Cmp(mul(t, den)) > 0 }

// rateBelow reports num/den < t (basis points), den > 0: a minimum
// fraction's failure.
func rateBelow(num, den, t int64) bool { return mul(num, 10_000).Cmp(mul(t, den)) < 0 }

// dropAbove reports nb/db − na/da > t (basis points), da, db > 0: the
// fall from b to a exceeds the tolerance.
func dropAbove(nb, db, na, da, t int64) bool {
	l := new(big.Int).Sub(mul(nb, da), mul(na, db))
	l.Mul(l, bi(10_000))
	return l.Cmp(mul(t, da, db)) > 0
}

// relDropAbove reports 1 − a/b > t (basis points), b > 0: a fell from b by
// more than the fraction.
func relDropAbove(a, b, t int64) bool { return mul(b-a, 10_000).Cmp(mul(t, b)) > 0 }

// growthAbove reports sum/n − ref/n > max(frac × ref/n, floor): the mean
// of n daily values grew past the tolerance (frac in basis points, floor
// in the values' unit).
func growthAbove(sum, ref, n, frac, floor int64) bool {
	l := mul(sum-ref, 10_000)
	a, b := mul(frac, ref), mul(floor, n, 10_000)
	if a.Cmp(b) < 0 {
		a = b
	}
	return l.Cmp(a) > 0
}

// failLimit is the most of n seeds allowed to fail at a fraction given in
// basis points: floor(fraction × n).
func failLimit(fracBP int64, n int) int { return int(fracBP * int64(n) / 10_000) }
