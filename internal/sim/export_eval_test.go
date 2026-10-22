package sim

import (
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/rng"
)

// NewEvalRecorder starts recording at the takeover instant; the world must
// be exactly there, with an open-ended evaluation window. commitmentDays
// is the policy's finance.employeeCommitmentDays. A holdout seed is
// refused (eval.GuardRepo): it is recorded only by NewHoldoutRecorder.
func (w *World) NewEvalRecorder(plan, planSHA256, fingerprint string, seed int64, commitmentDays int) *EvalRecorder {
	return w.NewEvalRecorderAt("../..", plan, planSHA256, fingerprint, seed, commitmentDays)
}

// NewEvalRecorderAt is NewEvalRecorder guarded by the holdout manifest of
// the repository at root: a test of the guard uses a synthetic one.
func (w *World) NewEvalRecorderAt(root, plan, planSHA256, fingerprint string, seed int64, commitmentDays int) *EvalRecorder {
	if err := eval.GuardRepo(root, []int64{seed}); err != nil {
		panic(err)
	}
	return w.newEvalRecorder(plan, planSHA256, fingerprint, seed, commitmentDays)
}

// NewEvalWorld is New for an evaluation harness: the seed is refused
// (eval.GuardRepo) before any world is built, so a holdout seed never
// reaches a world outside the canonical runner, whose seeds come only
// from an authorized protocol.
func NewEvalWorld(cfg *Config, seed int64) *World {
	return NewEvalWorldAt("../..", cfg, seed)
}

// NewEvalWorldAt is NewEvalWorld guarded by the holdout manifest of the
// repository at root.
func NewEvalWorldAt(root string, cfg *Config, seed int64) *World {
	if err := eval.GuardRepo(root, []int64{seed}); err != nil {
		panic(err)
	}
	return New(cfg, rng.SeedFromInt(seed))
}

// NewHoldoutRecorder is NewEvalRecorder for the canonical holdout runner:
// the seed must be one of an authorized protocol's (eval.Authorize).
func (w *World) NewHoldoutRecorder(p *eval.Protocol, plan, planSHA256, fingerprint string, seed int64) *EvalRecorder {
	if !p.Covers(seed) {
		panic("sim: the holdout recorder needs an authorized protocol covering the seed")
	}
	return w.newEvalRecorder(plan, planSHA256, fingerprint, seed, p.Policy.Finance.EmployeeCommitmentDays)
}
