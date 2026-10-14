package rng

import (
	"errors"
	"fmt"
)

// QuantilePoint is one knot of a piecewise-linear inverse CDF: the value at
// percentile P (in PPM).
type QuantilePoint struct {
	P PPM
	V int64
}

// Quantile is a piecewise-linear inverse CDF over integers, used for
// durations. Knots run from P=0 to P=1,000,000 with non-decreasing values.
type Quantile struct {
	pts []QuantilePoint
}

// NewQuantile validates and builds a table. Percentiles must start at 0, end
// at 1,000,000 and strictly increase; values must not decrease.
func NewQuantile(pts []QuantilePoint) (Quantile, error) {
	if len(pts) < 2 {
		return Quantile{}, errors.New("rng: quantile table needs at least p0 and p100")
	}
	if pts[0].P != 0 || pts[len(pts)-1].P != OneMillion {
		return Quantile{}, errors.New("rng: quantile table must start at p0 and end at p100")
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].P <= pts[i-1].P {
			return Quantile{}, fmt.Errorf("rng: quantile percentiles not increasing at knot %d", i)
		}
		if pts[i].V < pts[i-1].V {
			return Quantile{}, fmt.Errorf("rng: quantile values decrease at knot %d", i)
		}
	}
	return Quantile{pts: append([]QuantilePoint(nil), pts...)}, nil
}

// Min returns the p0 value.
func (q Quantile) Min() int64 { return q.pts[0].V }

// Max returns the p100 value; no sample exceeds it.
func (q Quantile) Max() int64 { return q.pts[len(q.pts)-1].V }

// At returns the inverse CDF at percentile u in [0, 1,000,000):
// va + floor((vb-va) * (u-pa) / (pb-pa)) on the segment containing u.
func (q Quantile) At(u PPM) int64 {
	pts := q.pts
	i := 1
	for i < len(pts)-1 && u >= pts[i].P {
		i++
	}
	a, b := pts[i-1], pts[i]
	return a.V + (b.V-a.V)*int64(u-a.P)/int64(b.P-a.P)
}

// Sample draws one value (one draw).
func (q Quantile) Sample(src Source) int64 {
	return q.At(PPM(Uniform(src, uint64(OneMillion))))
}
