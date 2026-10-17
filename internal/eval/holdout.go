package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const HoldoutSchema = "dreamteam.holdout/v1"

// Holdout is the locked holdout's manifest. It commits to the seeds by
// their method and digest; the protocol says when they may be run.
type Holdout struct {
	Schema   string   `json:"schema"`
	ID       string   `json:"id"`
	Method   string   `json:"method"`
	Label    string   `json:"label"`
	Count    int      `json:"count"`
	Lo       int64    `json:"lo"`
	Hi       int64    `json:"hi"`
	Excluded []string `json:"excludedSets"` // earlier seed sets, as ranges
	// SeedsSHA256: SHA-256 of the seeds in generation order, decimal and
	// comma-separated.
	SeedsSHA256 string   `json:"seedsSHA256"`
	Sealed      bool     `json:"sealed"`
	Protocol    []string `json:"protocol"`
	Unseal      string   `json:"unsealRecord"` // the file that, once committed, unseals it
}

// HoldoutSeeds generates the method's seeds: the i-th candidate (i = 0,
// 1, …) is lo + (the first 8 bytes of SHA-256("<label>/<i>") as a
// big-endian integer) mod (hi − lo + 1); a candidate already drawn or in
// an excluded set is skipped, until count seeds are drawn.
func HoldoutSeeds(label string, count int, lo, hi int64, excluded []string) ([]int64, error) {
	ex, err := parseRanges(excluded)
	if err != nil {
		return nil, err
	}
	if hi < lo || count <= 0 {
		return nil, errors.New("empty range")
	}
	span := uint64(hi - lo + 1)
	seen := map[int64]bool{}
	var out []int64
	for i := 0; len(out) < count; i++ {
		if i > 100*count {
			return nil, errors.New("range too small for the count")
		}
		sum := sha256.Sum256([]byte(label + "/" + strconv.Itoa(i)))
		s := lo + int64(binary.BigEndian.Uint64(sum[:8])%span)
		if seen[s] || ex(s) {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, nil
}

func seedsDigest(seeds []int64) string {
	parts := make([]string, len(seeds))
	for i, s := range seeds {
		parts[i] = strconv.FormatInt(s, 10)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, ",")))
	return hex.EncodeToString(sum[:])
}

func parseRanges(rs []string) (func(int64) bool, error) {
	type rg struct{ a, b int64 }
	var xs []rg
	for _, r := range rs {
		a, b, ok := strings.Cut(r, "-")
		lo, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("range %q", r)
		}
		hi := lo
		if ok {
			if hi, err = strconv.ParseInt(b, 10, 64); err != nil {
				return nil, fmt.Errorf("range %q", r)
			}
		}
		xs = append(xs, rg{lo, hi})
	}
	return func(s int64) bool {
		for _, x := range xs {
			if s >= x.a && s <= x.b {
				return true
			}
		}
		return false
	}, nil
}

// LoadHoldout reads a manifest and regenerates its seeds, which must
// match its digest.
func LoadHoldout(path string) (*Holdout, []int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var h Holdout
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&h); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	if h.Schema != HoldoutSchema {
		return nil, nil, fmt.Errorf("%s: schema %q", path, h.Schema)
	}
	seeds, err := HoldoutSeeds(h.Label, h.Count, h.Lo, h.Hi, h.Excluded)
	if err != nil {
		return nil, nil, err
	}
	if d := seedsDigest(seeds); d != h.SeedsSHA256 {
		return nil, nil, fmt.Errorf("%s: seeds digest %s, manifest %s", path, d, h.SeedsSHA256)
	}
	return &h, seeds, nil
}

// ErrSealed refuses a run that would touch the holdout outside the
// canonical runner.
var ErrSealed = errors.New("the holdout is sealed")

// GuardSeeds refuses any holdout seed. Sealed or not, a holdout seed runs
// only through the canonical runner, which holds an authorized Protocol
// (Authorize); every other harness calls this before running anything.
func GuardSeeds(seeds []int64, h *Holdout, holdout []int64) error {
	in := map[int64]bool{}
	for _, s := range holdout {
		in[s] = true
	}
	n := 0
	for _, s := range seeds {
		if in[s] {
			n++
		}
	}
	if n > 0 {
		// The seeds themselves are not named: the manifest commits to them
		// by digest only.
		return fmt.Errorf("%w: %d of the seeds are in %s, which runs only through the canonical holdout runner", ErrSealed, n, h.ID)
	}
	return nil
}

// Repository paths of the frozen evaluation: the policy in force, the
// superseded one (kept byte for byte) and the holdout manifest.
const (
	PolicyPath   = "docs/evaluation/policy-v0.2-eval-2.json"
	PolicyV1Path = "docs/evaluation/policy-v0.2.json"
	HoldoutPath  = "docs/evaluation/holdout-v0.2.json"
)

// GuardRepo is GuardSeeds with the repository's holdout, from the
// repository root.
func GuardRepo(root string, seeds []int64) error {
	h, hs, err := LoadHoldout(root + "/" + HoldoutPath)
	if err != nil {
		return err
	}
	return GuardSeeds(seeds, h, hs)
}
