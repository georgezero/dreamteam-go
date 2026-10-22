package eval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Holdout containment (Slice 5C.4a), by reading the source, never by
// running it: the places that can reach the repository's holdout manifest
// or regenerate holdout seeds are exactly these. A new one fails here and
// must be reviewed. Build-tagged files (calibration, acceptance) are read
// too. Function names are "File:Func" (Func "" for a top-level
// declaration).
var holdoutReach = map[string][]string{
	// The manifest's path, by name.
	"HoldoutPath": {
		"internal/eval/holdout.go:",          // the constant
		"internal/eval/holdout.go:GuardRepo", // the guard: an error, never the seeds
		"internal/eval/protocol.go:",         // RepoPaths
		"internal/eval/eval_test.go:TestHoldoutSealed",
		"internal/eval/eval_test.go:synthHoldout",             // a synthetic manifest at that path under a new root
		"internal/eval/eval_test.go:TestAuthorizeSealedFirst", // rewrites that synthetic manifest
	},
	// Regenerating a manifest's seeds.
	"LoadHoldout": {
		"internal/eval/holdout.go:GuardRepo",
		"internal/eval/protocol.go:Authorize", // only once an unseal record exists
		"internal/eval/eval_test.go:TestHoldoutSealed",
		"internal/eval/eval_test.go:synthHoldout",
		"internal/sim/holdout_protocol_test.go:TestHoldoutRecordersGuarded", // the synthetic fixture's
	},
	// Regenerating seeds from a manifest's bytes: only the two readers.
	"decodeHoldout": {
		"internal/eval/holdout.go:LoadHoldout",
		"internal/eval/holdout.go:GuardManifest", // an error, never the seeds
	},
	// The guard on the game's embedded manifest: only the T90 evaluation,
	// which refuses a holdout seed before running anything.
	"GuardManifest": {
		"internal/session/t90.go:T90Job",
	},
	// The repository's own paths with the repository as root: only the
	// sealed checks, the canonical run and the synthetic fixture.
	"Authorize:RepoPaths": {
		"internal/eval/eval_test.go:TestRepositoryProtocol",
		"internal/sim/holdout_runner_test.go:TestHoldoutRun",
		"internal/sim/holdout_protocol_test.go:authorizeAs",   // the synthetic fixture's root
		"internal/eval/eval_test.go:TestAuthorizeSealedFirst", // synthHoldout's root
	},
}

// HoldoutSeeds outside LoadHoldout takes only a synthetic label (the
// repository's holdout label is never spelled in source).
const synthLabelPrefix = "dreamteam/test/"

func TestHoldoutContainment(t *testing.T) {
	found := map[string][]string{}
	var badLabels []string
	fset := token.NewFileSet()
	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("../..", path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "detail" || rel == ".git" || strings.HasPrefix(d.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			fn := ""
			if fd, ok := decl.(*ast.FuncDecl); ok {
				fn = fd.Name.Name
			}
			where := rel + ":" + fn
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					if x.Name == "HoldoutPath" {
						found["HoldoutPath"] = append(found["HoldoutPath"], where)
					}
				case *ast.CallExpr:
					switch name := callName(x); name {
					case "LoadHoldout", "decodeHoldout", "GuardManifest":
						found[name] = append(found[name], where)
					case "Authorize":
						for _, a := range x.Args {
							if callName(&ast.CallExpr{Fun: a}) == "RepoPaths" {
								found["Authorize:RepoPaths"] = append(found["Authorize:RepoPaths"], where)
							}
						}
					case "HoldoutSeeds":
						if where == "internal/eval/holdout.go:decodeHoldout" || where == "internal/eval/eval_test.go:TestGuardSeeds" {
							break // the manifest's own label; a synthetic manifest's label, altered
						}
						if lit, ok := x.Args[0].(*ast.BasicLit); ok {
							if s, _ := strconv.Unquote(lit.Value); strings.HasPrefix(s, synthLabelPrefix) {
								break
							}
						}
						if sel, ok := x.Args[0].(*ast.SelectorExpr); ok && where == "internal/eval/eval_test.go:synthHoldout" && sel.Sel.Name == "Label" {
							break
						}
						badLabels = append(badLabels, where)
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for kind, allowed := range holdoutReach {
		for _, w := range found[kind] {
			if !slices.Contains(allowed, w) {
				t.Errorf("%s reached from %s, which is not reviewed (holdoutReach)", kind, w)
			}
		}
		for _, a := range allowed {
			if !slices.Contains(found[kind], a) {
				t.Errorf("%s: reviewed site %s no longer reaches it; update holdoutReach", kind, a)
			}
		}
	}
	for _, w := range badLabels {
		t.Errorf("HoldoutSeeds with a label that is not synthetic (%q...) in %s", synthLabelPrefix, w)
	}
}

// callName is the called function's name: f(...), pkg.f(...) or x.f(...).
func callName(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}
