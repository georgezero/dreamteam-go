package scenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/scenarios"
)

// Default is the scenario played when none is named: the established
// practice, built by its 180-day warm-up (ROADMAP §2, step 11).
const Default = "established"

// List returns the IDs of the bundled playable scenarios.
func List() []string {
	entries, _ := fs.ReadDir(scenarios.FS, ".")
	var ids []string
	for _, e := range entries {
		if n := e.Name(); !e.IsDir() && strings.HasSuffix(n, ".json") {
			ids = append(ids, strings.TrimSuffix(n, ".json"))
		}
	}
	slices.Sort(ids)
	return ids
}

// Resolve loads and compiles a bundled scenario by ID, or a scenario file by
// path (anything ending in .json). A file's extends and preset resolve
// against its own directory first, then against the bundled scenarios.
func Resolve(ref string) (*sim.Config, error) {
	f, err := ResolveFile(ref)
	if err != nil {
		return nil, err
	}
	return Compile(f)
}

// ResolveFile loads a scenario (bundled ID or .json path) with extends and
// preset merged: the fully resolved file a session record embeds.
func ResolveFile(ref string) (*File, error) {
	fsys, name := fs.FS(scenarios.FS), ref+".json"
	if strings.HasSuffix(ref, ".json") {
		dir, base := filepath.Split(ref)
		if dir == "" {
			dir = "."
		}
		fsys, name = overlayFS{os.DirFS(dir), scenarios.FS}, base
	}
	return Load(fsys, name)
}

// Canonical is a resolved scenario's canonical JSON (struct field order,
// map keys sorted): what records embed and fingerprints hash.
func Canonical(f *File) ([]byte, error) { return json.Marshal(f) }

// Fingerprint is the SHA-256 of a resolved scenario's canonical JSON.
func Fingerprint(f *File) (string, error) {
	raw, err := Canonical(f)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// Decode reads a resolved scenario from its canonical JSON, strictly.
func Decode(raw []byte) (*File, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f File
	if err := dec.Decode(&f); err != nil {
		return nil, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("scenario: data after the scenario")
	}
	return &f, nil
}

// overlayFS reads from top, falling back to bottom.
type overlayFS struct{ top, bottom fs.FS }

func (o overlayFS) Open(name string) (fs.File, error) {
	f, err := o.top.Open(name)
	if err == nil {
		return f, nil
	}
	return o.bottom.Open(name)
}
