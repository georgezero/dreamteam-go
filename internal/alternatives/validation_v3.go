package alternatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

const ParserV3ID = "dreamteam.phase-response-parser/v3"

// The authoritative decoder supplies every protocol shape and size constraint;
// the handler callback supplies duplicate-target and session validation.
func protocolValidatorV3(v Validator) Validator {
	return func(actions []api.Action) ([]api.ActionResult, error) {
		raw, err := json.Marshal(struct {
			Actions []api.Action `json:"actions"`
		}{actions})
		if err != nil {
			return nil, err
		}
		decoded, err := agent.DecodeBatch(raw)
		if err != nil {
			return nil, err
		}
		return v(decoded)
	}
}

func DecideV3With(s *session.Session, v Validator, arm string, proposal []api.Action, inspected []string, r *Reply) (*OutcomeV2, error) {
	out, err := DecideV2With(s, protocolValidatorV3(v), arm, proposal, inspected, r)
	if err != nil {
		return nil, err
	}
	out.Schema = "dreamteam.alternatives-outcome/v3"
	out.SubmittedAction = CanonicalSubmittedV3(out.Batches, s.Calendar().Date(s.Now()).String())
	return out, nil
}

// Normalize only integer-typed action fields. UseNumber and exact rational
// arithmetic preserve precision; strings and fractional/overflow values still
// reach the strict typed decoder and are refused. Citation values are untouched.
func normalizeActionNumbers(raw []byte) ([]byte, error) {
	if err := exactJSON(raw, reflect.TypeOf(replyWire{})); err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	a, ok := obj["actions"]
	if !ok {
		return raw, nil
	}
	dec := json.NewDecoder(bytes.NewReader(a))
	dec.UseNumber()
	var actions any
	if err := dec.Decode(&actions); err != nil {
		return nil, err
	}
	var walk func(any, reflect.Type) (any, error)
	walk = func(value any, t reflect.Type) (any, error) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch x := value.(type) {
		case json.Number:
			if t.Kind() != reflect.Int64 && t.Kind() != reflect.Int {
				return x, nil
			}
			text := string(x)
			// Bound exponent expansion before exact rational parsing. Nonzero values
			// outside this bound cannot be integral int64 values. Zero never expands.
			mantissa, exponent, has := strings.Cut(strings.ToLower(text), "e")
			if strings.Trim(mantissa, "-0.") == "" {
				return json.Number("0"), nil
			}
			if has {
				exp, err := strconv.ParseInt(exponent, 10, 64)
				if err != nil || exp > int64(len(mantissa)+20) || exp < -int64(len(mantissa)+20) {
					return nil, fmt.Errorf("action integer out of range")
				}
			}
			rat, ok := new(big.Rat).SetString(text)
			if !ok || !rat.IsInt() || !rat.Num().IsInt64() {
				return nil, fmt.Errorf("action number must be an in-range integer")
			}
			return json.Number(rat.Num().String()), nil
		case map[string]any:
			if t.Kind() != reflect.Struct {
				return value, nil
			}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				name := strings.Split(field.Tag.Get("json"), ",")[0]
				if v, ok := x[name]; ok {
					n, err := walk(v, field.Type)
					if err != nil {
						return nil, err
					}
					x[name] = n
				}
			}
		case []any:
			if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
				return value, nil
			}
			for i, v := range x {
				n, err := walk(v, t.Elem())
				if err != nil {
					return nil, err
				}
				x[i] = n
			}
		}
		return value, nil
	}
	value, err := walk(actions, reflect.TypeOf([]api.Action{}))
	if err != nil {
		return nil, err
	}
	obj["actions"], err = json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

func ParseTurnReplyV3(content string) TurnReply {
	raw, err := normalizeActionNumbers([]byte(content))
	if err != nil {
		return TurnReply{Kind: ReplyUnreadable, Error: err.Error()}
	}
	return ParseTurnReplyV2(string(raw))
}
