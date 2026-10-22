// Package evaluation embeds the frozen evaluation files the game reads at
// run time: the policy in force and the holdout manifest, byte for byte as
// committed beside this file.
package evaluation

import _ "embed"

// PolicyV4 is policy-v0.2-eval-4.json, the evaluation policy in force.
//
//go:embed policy-v0.2-eval-4.json
var PolicyV4 []byte

// Holdout is holdout-v0.2.json, the sealed holdout manifest (seeds by
// digest only).
//
//go:embed holdout-v0.2.json
var Holdout []byte
