package agent

import (
	"encoding/json"
	"github.com/georgezero/dreamteam-go/internal/api"
)

// DecodeBatch exposes the authoritative protocol decoder unchanged. Successor
// read-only extensions use it before the existing duplicate/session validator.
func DecodeBatch(raw json.RawMessage) ([]api.Action, error) {
	actions, err := batch(raw)
	if err != nil {
		return nil, err
	}
	return actions, nil
}
