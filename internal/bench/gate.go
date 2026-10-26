package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/agent"
)

// PolicyOps are the protocol operations a policy may call: reading the
// session and checking batches. apply, advance, save and t90 are the
// harness's: it applies the batches a policy returns, moves time by the
// configured cadence, saves, and evaluates after the run (t90 simulates
// the rest of the horizon with the run's seed, which no policy may see
// while it decides).
var PolicyOps = []string{"hello", "observe", "sections", "actions", "validate", "preview", "digests"}

// ErrNotPermitted is the gate's error code for an operation a policy may
// not call. It is a harness code, not one of the protocol's: the request
// never reaches the session.
const ErrNotPermitted = "not_permitted"

// gate stands between a policy's client and the session's handler. It
// reads each request's op the way the handler does (the same decoder,
// unknown fields refused) and passes a permitted request's bytes on
// unchanged; a request it cannot read goes on too, for the handler to
// refuse. It answers anything else itself, in the protocol's response
// format, and counts it.
type gate struct {
	h      agent.Exchanger
	denied int
}

func (g *gate) Handle(line []byte) []byte {
	var req agent.Request
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if !json.Valid(line) || dec.Decode(&req) != nil || req.Op == "" || slices.Contains(PolicyOps, req.Op) {
		return g.h.Handle(line)
	}
	id := json.RawMessage("null")
	if len(req.ID) > 0 {
		var v any
		json.Unmarshal(req.ID, &v)
		switch v.(type) {
		case string, float64, nil:
			var b bytes.Buffer
			json.Compact(&b, req.ID)
			id = b.Bytes()
		default: // an id the handler refuses
			return g.h.Handle(line)
		}
	}
	g.denied++
	out, _ := json.Marshal(agent.Response{Protocol: agent.Protocol, ID: id, Error: &agent.Error{Code: ErrNotPermitted,
		Message: fmt.Sprintf("a benchmark policy may call %v; %q is the harness's", PolicyOps, req.Op), Field: "op"}})
	return out
}
