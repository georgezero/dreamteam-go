package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Development: a strategy set run on the policy's development seeds
// (Slice 5C.1, docs/calibration/strategy-development). The set and its
// strategy files follow the holdout contract (strict decoding, every file
// at its pinned digest), so a frozen set needs no conversion; but nothing
// here unseals anything. The protocol it returns is never authorized: it
// covers no holdout seed, and NewHoldoutRecorder refuses it.

// DevelopmentSeeds are the policy's development seeds ("901-910").
func DevelopmentSeeds(p *Policy) ([]int64, error) {
	lo, hi, ok := strings.Cut(p.Seeds.Development, "-")
	a, err1 := strconv.ParseInt(lo, 10, 64)
	b, err2 := strconv.ParseInt(hi, 10, 64)
	if !ok || err1 != nil || err2 != nil || a > b {
		return nil, fmt.Errorf("policy development seeds %q are not a range", p.Seeds.Development)
	}
	var out []int64
	for s := a; s <= b; s++ {
		out = append(out, s)
	}
	return out, nil
}

// Development loads a strategy set (repository-relative setPath) for a
// development run on seeds, which must all be the policy's development
// seeds: never a reference or a holdout seed.
func Development(root, setPath string, p *Policy, digest string, seeds []int64) (*Protocol, string, error) {
	dev, err := DevelopmentSeeds(p)
	if err != nil {
		return nil, "", err
	}
	for _, s := range seeds {
		if !slices.Contains(dev, s) {
			return nil, "", fmt.Errorf("seed %d is not a development seed (%s)", s, p.Seeds.Development)
		}
	}
	if err := GuardRepo(root, seeds); err != nil {
		return nil, "", err
	}
	at := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	raw, err := os.ReadFile(at(setPath))
	if err != nil {
		return nil, "", err
	}
	pr := &Protocol{Root: root, Seeds: seeds, Policy: p, PolicyDigest: digest, Strategies: map[string]Strategy{}, StrategySHA: map[string]string{}}
	set := &pr.Set
	if err := decodeStrict(raw, set); err != nil || set.Schema != StrategySetSchema {
		return nil, "", fmt.Errorf("strategy set %s: %v (schema %q)", setPath, err, set.Schema)
	}
	for _, x := range set.Plans {
		if _, dup := pr.Strategies[x.Name]; dup || x.Name == "" {
			return nil, "", fmt.Errorf("strategy %q twice or unnamed", x.Name)
		}
		sraw, err := os.ReadFile(at(x.Path))
		if err != nil {
			return nil, "", fmt.Errorf("strategy %s: %v", x.Name, err)
		}
		if sha(sraw) != x.SHA256 {
			return nil, "", fmt.Errorf("strategy %s (%s) is not at its pinned digest", x.Name, x.Path)
		}
		var st Strategy
		if err := decodeStrict(sraw, &st); err != nil || st.Schema != StrategySchema || st.Name != x.Name {
			return nil, "", fmt.Errorf("strategy %s: %v (schema %q, name %q)", x.Path, err, st.Schema, st.Name)
		}
		pr.Strategies[x.Name], pr.StrategySHA[x.Name] = st, x.SHA256
	}
	switch {
	case len(set.Plans) < 2:
		return nil, "", fmt.Errorf("the strategy set needs the baseline and at least one candidate")
	case pr.Strategies[set.Baseline].Name == "":
		return nil, "", fmt.Errorf("baseline %q is not in the strategy set", set.Baseline)
	case len(pr.Strategies[set.Baseline].Batches) != 0:
		return nil, "", fmt.Errorf("baseline %q takes actions", set.Baseline)
	}
	for _, st := range pr.Strategies {
		if st.TeleOnlyOf != "" && (st.TeleOnlyOf == st.Name || pr.Strategies[st.TeleOnlyOf].Name == "") {
			return nil, "", fmt.Errorf("strategy %s: teleOnlyOf %q is not another strategy of the set", st.Name, st.TeleOnlyOf)
		}
	}
	return pr, sha(raw), nil
}
