package alternatives

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/api"
)

func TestV13gBatchProtocolAgreement(t *testing.T) {
	p := []api.Action{term("E04", "2026-10-27")}
	op := api.Action{Type: api.ActSetWalkInPolicy, Policy: "schedule_later"}
	for _, n := range []int{0, 1, 64, 65} {
		for _, mixed := range []bool{false, true} {
			s := at(t, t0)
			actions := make([]api.Action, n)
			for i := range actions {
				actions[i] = op
			}
			if mixed {
				actions = append(actions, p...)
			}
			before := s.Digests()
			out, err := DecideV3With(s, SessionValidator(s), ArmControl, p, nil, &Reply{Kind: ReplyRevise, Actions: actions})
			if err != nil {
				t.Fatal(err)
			}
			if out.Valid != (n <= 64) {
				t.Fatalf("n=%d mixed=%v valid=%v", n, mixed, out.Valid)
			}
			if s.Digests() != before {
				t.Fatal("validation mutated session")
			}
			if n > 64 && len(out.Batches) != 0 {
				t.Fatal("known invalid operations submitted staffing")
			}
			h := agent.New(s)
			for _, b := range out.Batches {
				raw, _ := json.Marshal(map[string]any{"op": "apply", "args": map[string]any{"actions": b}})
				var result struct {
					OK     bool `json:"ok"`
					Result struct {
						Applied bool `json:"applied"`
					} `json:"result"`
				}
				if err := json.Unmarshal(h.Handle(raw), &result); err != nil {
					t.Fatal(err)
				}
				if !result.OK || !result.Result.Applied {
					t.Fatalf("validated batch refused on apply: %s", h.Handle(raw))
				}
			}
		}
	}
	for _, actions := range [][]api.Action{{{Type: "unknown"}}, {{Type: api.ActSetWalkInPolicy, Policy: "unknown"}, p[0]}} {
		s := at(t, t0)
		out, err := DecideV3With(s, SessionValidator(s), ArmControl, p, nil, &Reply{Kind: ReplyRevise, Actions: actions})
		if err != nil || out.Valid || len(out.Batches) > 0 {
			t.Fatal(err, out)
		}
	}
}

func TestV13gNumericSchema(t *testing.T) {
	for _, n := range []string{"30", "30.0", "3e1", "300e-1", "-0", "9007199254740993", "9223372036854775807", "-9223372036854775808"} {
		r := ParseTurnReplyV3(fmt.Sprintf(`{"actions":[{"type":"extend_shift","employee":"E03","minutes":%s}]}`, n))
		if r.Kind != ReplyRevise {
			t.Fatalf("%s: %+v", n, r)
		}
	}
	for _, n := range []string{"30.1", "9223372036854775808", "-9223372036854775809", "1e309", "1e-99999", "NaN", "Infinity", `"30"`, `"bad"`} {
		r := ParseTurnReplyV3(fmt.Sprintf(`{"actions":[{"type":"extend_shift","employee":"E03","minutes":%s}]}`, n))
		if r.Kind != ReplyUnreadable {
			t.Fatalf("accepted %s", n)
		}
	}
	for _, raw := range []string{`{"actions":[{"type":"extend_shift","minutes":30.0,"extra":1}]}`, `{"actions":[{"type":"extend_shift","minutes":30.0,"minutes":31}]}`} {
		if ParseTurnReplyV3(raw).Kind != ReplyUnreadable {
			t.Fatal("accepted unknown or duplicate field")
		}
	}
}

func TestV13gCanonicalPrecisionAndCollections(t *testing.T) {
	a := api.Action{Type: api.ActSetWalkInPolicy, Policy: "schedule_later", Minutes: 9007199254740992}
	b := a
	b.Minutes++
	if CanonicalSubmittedV3([][]api.Action{{a}}).Sha256 == CanonicalSubmittedV3([][]api.Action{{b}}).Sha256 {
		t.Fatal("integer collision")
	}
	empty := CanonicalSubmittedV3(nil)
	if empty.Canonical != "[]" || empty.Sha256 != CanonicalSubmitted(nil).Sha256 {
		t.Fatal(empty)
	}
	raw := `{"type":"set_roster","employee":"E03","op":"set","effective":"2026-11-02","roster":{"A":{"Mon":{"start":"08:00","end":"12:00","assignment":"R1","breaks":[],"covers":[]}},"B":{}}}`
	var identities []string
	for _, v := range []string{raw, strings.ReplaceAll(strings.ReplaceAll(raw, `"breaks":[]`, `"breaks":null`), `"covers":[]`, `"covers":null`), strings.ReplaceAll(raw, `,"breaks":[],"covers":[]`, "")} {
		var act api.Action
		if err := json.Unmarshal([]byte(v), &act); err != nil {
			t.Fatal(err)
		}
		identities = append(identities, CanonicalSubmittedV3([][]api.Action{{act}}).Sha256)
	}
	if identities[0] != identities[1] || identities[1] != identities[2] {
		t.Fatal(identities)
	}
	ext := api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 30}
	dated := ext
	dated.Date = "2026-10-13"
	if CanonicalSubmittedV3([][]api.Action{{ext}}, dated.Date).Sha256 != CanonicalSubmittedV3([][]api.Action{{dated}}, dated.Date).Sha256 {
		t.Fatal("default date")
	}
	different := dated
	different.Date = "2026-10-14"
	other := ext
	other.Employee = "E04"
	for _, acts := range [][][]api.Action{{{different}}, {{other}}, {{ext, other}}, {{other, ext}}} {
		if CanonicalSubmittedV3(acts, dated.Date).Sha256 == CanonicalSubmittedV3([][]api.Action{{dated}}, dated.Date).Sha256 {
			t.Fatal("distinct submission collapsed")
		}
	}
	if CanonicalSubmittedV3([][]api.Action{{ext, other}}, dated.Date).Sha256 == CanonicalSubmittedV3([][]api.Action{{other, ext}}, dated.Date).Sha256 {
		t.Fatal("order collapsed")
	}
}
