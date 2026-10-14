package scenario

import (
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
	fsys, name := fs.FS(scenarios.FS), ref+".json"
	if strings.HasSuffix(ref, ".json") {
		dir, base := filepath.Split(ref)
		if dir == "" {
			dir = "."
		}
		fsys, name = overlayFS{os.DirFS(dir), scenarios.FS}, base
	}
	f, err := Load(fsys, name)
	if err != nil {
		return nil, err
	}
	return Compile(f)
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
