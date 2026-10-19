package usecase

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

// ApplyDelta applies an RFC 6902 JSON Patch to a base payload.
// The result passes the same gates as a full form.
func ApplyDelta(base, ops json.RawMessage) (json.RawMessage, error) {
	if len(base) == 0 {
		return nil, fmt.Errorf("apply delta: base is empty")
	}
	if len(ops) == 0 {
		return nil, fmt.Errorf("apply delta: ops is empty")
	}
	patch, err := jsonpatch.DecodePatch(ops)
	if err != nil {
		return nil, fmt.Errorf("apply delta: decode patch: %w", err)
	}
	out, err := patch.Apply(base)
	if err != nil {
		return nil, fmt.Errorf("apply delta: %w", err)
	}
	return out, nil
}
