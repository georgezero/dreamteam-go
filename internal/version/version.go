// Package version identifies the running build. Number advances with each
// roadmap step (0.1.0-dev.N while V0.1 step N is the latest; a further
// .M marks a remediation pass on that step), so a player can tell which
// step a binary contains.
package version

import "runtime/debug"

// Number is the release number.
const Number = "0.1.0-dev.14.1"

// String returns the version with the source commit when the binary was
// built from a git checkout ("0.1.0-dev.8 (72b9c91)", "+dirty" when the
// tree had uncommitted changes). go run builds carry no commit.
func String() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Number
	}
	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if rev == "" {
		return Number
	}
	if dirty {
		rev += "+dirty"
	}
	return Number + " (" + rev + ")"
}
