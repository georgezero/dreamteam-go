package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/session"
)

// Check is one verification step's outcome.
type Check struct {
	What string `json:"what"`
	OK   bool   `json:"ok"`
	Note string `json:"note,omitempty"`
}

// VerifyOptions: T90 also recomputes each evaluated run's T90 from its
// saved session (two simulations per run) and compares it with t90.json.
type VerifyOptions struct {
	T90 bool
}

// Verify checks a results directory's integrity: every file's SHA-256
// against the manifest, and nothing else in it; each saved session
// replayed (every digest verified) and its canonical result rebuilt byte
// for byte; each run record's metrics read again from that result; the
// session's batches against the recorded decisions; and the report
// rebuilt from the run records. It checks that these files agree with
// each other and with the simulator; it does not certify them
// independently, and a run is checked only as far as its files go.
func Verify(dir string, opts VerifyOptions) ([]Check, error) {
	m, c, recs, err := LoadResults(dir)
	if err != nil {
		return nil, err
	}
	var out []Check
	add := func(what string, err error) {
		ck := Check{What: what, OK: err == nil}
		if err != nil {
			ck.Note = err.Error()
		}
		out = append(out, ck)
	}
	listed := []string{FileManifest, m.Config.Path}
	listed = append(listed, m.Nondeterministic...)
	hashes := []FileHash{m.Config}
	hashes = append(hashes, m.Reports...)
	for _, r := range m.Runs {
		hashes = append(hashes, r.Files...)
	}
	for _, h := range hashes {
		got, err := hashFile(dir, h.Path)
		if err == nil && got.SHA256 != h.SHA256 {
			err = fmt.Errorf("SHA-256 %s, manifest %s", got.SHA256, h.SHA256)
		}
		add("hash "+h.Path, err)
		listed = append(listed, h.Path)
	}
	var extra []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			if rel = filepath.ToSlash(rel); !slices.Contains(listed, rel) {
				extra = append(extra, rel)
			}
		}
		return nil
	})
	if len(extra) > 0 {
		add("no unlisted files", fmt.Errorf("not in the manifest: %v", extra))
	} else {
		add("no unlisted files", nil)
	}

	for i, rec := range recs {
		rd := filepath.Join(dir, "runs", rec.ID)
		if rec.ID != m.Runs[i].ID || rec.Status != m.Runs[i].Status {
			add(rec.ID+": record matches the manifest", fmt.Errorf("run.json says %s %s", rec.ID, rec.Status))
		}
		if _, err := os.Stat(filepath.Join(rd, FileSession)); err != nil {
			add(rec.ID+": saved session", fmt.Errorf("no session file (status %s)", rec.Status))
			continue
		}
		res, rep, err := Measure(filepath.Join(rd, FileSession))
		if err == nil && rec.Final != nil && rep.Replayed != rec.Final.Digests {
			err = fmt.Errorf("replayed run digest %s, recorded %s", rep.Replayed.Run, rec.Final.Digests.Run)
		}
		add(rec.ID+": session replays, every digest verified", err)
		if err != nil {
			continue
		}
		stored, err := os.ReadFile(filepath.Join(rd, FileResult))
		if err == nil && !bytes.Equal(stored, res) {
			err = fmt.Errorf("%s differs from the replayed session's result", FileResult)
		}
		add(rec.ID+": result.json is the replayed session's result", err)
		vals, err := Extract(res)
		if err == nil {
			a, _ := json.Marshal(vals)
			b, _ := json.Marshal(rec.Metrics)
			if !bytes.Equal(a, b) {
				err = fmt.Errorf("run.json metrics differ from the result's values")
			}
		}
		add(rec.ID+": metrics are the result's values", err)
		var decisions []Decision
		err = readJSON(filepath.Join(rd, FileDecisions), &decisions)
		if err == nil {
			var raw []byte
			if raw, err = os.ReadFile(filepath.Join(rd, FileSession)); err == nil {
				err = MatchSession(raw, decisions)
			}
		}
		add(rec.ID+": session batches are the applied decisions", err)
		if opts.T90 && rec.T90.Status == T90Evaluated {
			add(rec.ID+": t90.json recomputed", verifyT90(rd))
		}
	}

	r := BuildReport(c, recs)
	want, err := json.MarshalIndent(r, "", " ")
	if err == nil {
		var got []byte
		if got, err = os.ReadFile(filepath.Join(dir, FileReportJSON)); err == nil && !bytes.Equal(got, append(want, '\n')) {
			err = fmt.Errorf("%s differs from the report of the run records", FileReportJSON)
		}
	}
	add("report.json rebuilt from the run records", err)
	got, err := os.ReadFile(filepath.Join(dir, FileReportText))
	if err == nil && string(got) != r.Text() {
		err = fmt.Errorf("%s differs from the report of the run records", FileReportText)
	}
	add("report.txt rebuilt from the run records", err)
	return out, nil
}

// verifyT90 recomputes the T90 evaluation of a run's saved session and
// compares it with t90.json's evaluation.
func verifyT90(rd string) error {
	s, _, err := session.Load(filepath.Join(rd, FileSession), session.ReplayOptions{})
	if err != nil {
		return err
	}
	t, err := s.T90()
	if err != nil {
		return err
	}
	var stored struct {
		Evaluation json.RawMessage `json:"evaluation"`
	}
	if err := readJSON(filepath.Join(rd, FileT90), &stored); err != nil {
		return err
	}
	want, err := json.Marshal(t)
	if err != nil {
		return err
	}
	var got bytes.Buffer
	if err := json.Compact(&got, stored.Evaluation); err != nil {
		return err
	}
	if !bytes.Equal(got.Bytes(), want) {
		return fmt.Errorf("%s differs from the recomputed evaluation", FileT90)
	}
	return nil
}
