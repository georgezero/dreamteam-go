package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/georgezero/dreamteam-go/internal/version"
)

// Reproduce reruns a results directory's recorded configuration (its
// config.json, verbatim) into out, which must not exist or be empty, and
// compares the new manifest with the recorded one: the same engine,
// protocol, policies (and an external policy's files) and metric
// definitions, and every deterministic file's SHA-256. The
// nondeterministic files (timing.json, an external policy's standard
// error) are not compared. An external policy's command is resolved from
// the directory Reproduce runs in, as benchmark run resolves it.
func Reproduce(dir, out string, progress io.Writer) ([]Check, error) {
	m, c, _, err := LoadResults(dir)
	if err != nil {
		return nil, err
	}
	if m.Engine != version.Number {
		return nil, fmt.Errorf("the results were recorded by engine %s; this is %s: a rerun is not expected to match", m.Engine, version.Number)
	}
	raw, err := os.ReadFile(filepath.Join(dir, FileConfig))
	if err != nil {
		return nil, err
	}
	n, err := Matrix(c, raw, out, progress)
	if err != nil {
		return nil, err
	}
	var checks []Check
	add := func(what string, ok bool, note string) {
		checks = append(checks, Check{What: what, OK: ok, Note: note})
	}
	same := func(a, b any) bool {
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		return bytes.Equal(x, y)
	}
	add("engine, protocol and policy operations", m.Engine == n.Engine && m.Protocol == n.Protocol && same(m.PolicyOps, n.PolicyOps), "")
	add("policies (and external policy files)", same(m.Policies, n.Policies), "a policy, its version or an external policy's file differs")
	add("metric definitions", same(m.Metrics, n.Metrics), "")
	recorded := map[string]string{m.Config.Path: m.Config.SHA256}
	var order []string
	note := func(h FileHash) {
		if _, ok := recorded[h.Path]; !ok {
			order = append(order, h.Path)
		}
	}
	for _, h := range m.Reports {
		recorded[h.Path] = h.SHA256
	}
	for _, r := range m.Runs {
		for _, h := range r.Files {
			recorded[h.Path] = h.SHA256
		}
	}
	rerun := map[string]string{n.Config.Path: n.Config.SHA256}
	order = append(order, m.Config.Path)
	for _, r := range m.Runs {
		for _, h := range r.Files {
			order = append(order, h.Path)
		}
	}
	for _, h := range m.Reports {
		order = append(order, h.Path)
	}
	for _, r := range n.Runs {
		for _, h := range r.Files {
			note(h)
			rerun[h.Path] = h.SHA256
		}
	}
	for _, h := range n.Reports {
		note(h)
		rerun[h.Path] = h.SHA256
	}
	for i, r := range m.Runs {
		ok := i < len(n.Runs) && n.Runs[i].ID == r.ID && n.Runs[i].Status == r.Status
		got := "missing"
		if i < len(n.Runs) {
			got = n.Runs[i].ID + " " + n.Runs[i].Status
		}
		add(r.ID+": status "+r.Status, ok, "rerun: "+got)
	}
	for _, p := range order {
		a, inA := recorded[p]
		b, inB := rerun[p]
		switch {
		case !inA:
			add("file "+p, false, "only in the rerun")
		case !inB:
			add("file "+p, false, "not in the rerun")
		default:
			add("file "+p, a == b, fmt.Sprintf("SHA-256 %s, recorded %s", b, a))
		}
	}
	x, err := os.ReadFile(filepath.Join(dir, FileManifest))
	if err != nil {
		return nil, err
	}
	y, err := os.ReadFile(filepath.Join(out, FileManifest))
	if err != nil {
		return nil, err
	}
	add("manifest.json byte-identical", bytes.Equal(x, y), "the manifests differ")
	for i := range checks {
		if checks[i].OK {
			checks[i].Note = ""
		}
	}
	return checks, nil
}
