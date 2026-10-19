package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CanonicalJSON returns a deterministic encoding of a JSON document.
// encoding/json sorts the keys of a map, so two documents with the same
// members in a different order produce the same bytes.
// shortcut: numbers pass through float64, so 1e2 and 100 encode the same.
// Upgrade when a form carries exact decimals that must not be normalized.
func CanonicalJSON(raw json.RawMessage) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("canonical json: %w", err)
	}
	return json.Marshal(value)
}

// HashPayload returns the sha256 of the canonical payload.
func HashPayload(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("hash payload: payload is empty")
	}
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
