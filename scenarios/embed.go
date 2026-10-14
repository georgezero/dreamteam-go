// Package scenarios embeds the bundled scenario files. Top-level files are
// playable scenarios; base/ holds files they extend.
package scenarios

import "embed"

// FS holds the bundled scenarios.
//
//go:embed *.json base/*.json
var FS embed.FS
