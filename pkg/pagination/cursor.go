package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Cursor represents a standard keyset pagination cursor that supports
// dynamic sorting as defined in LLD 2.2b.
type Cursor struct {
	SortMode  string          `json:"sort_mode"`
	SortValue json.RawMessage `json:"sort_value"` // e.g., string, time.Time, etc.
	LastID    uuid.UUID       `json:"last_id"`
}

// EncodeCursor converts the Cursor struct into a base64-encoded JSON string.
func EncodeCursor(c Cursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

// DecodeCursor parses a base64-encoded JSON string back into a Cursor struct.
func DecodeCursor(s string) (Cursor, error) {
	var c Cursor
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return c, fmt.Errorf("decode cursor string: %w", err)
	}

	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("unmarshal cursor json: %w", err)
	}

	return c, nil
}

// Helper methods to extract common SortValue types

func (c *Cursor) GetStringValue() (string, error) {
	var val string
	if err := json.Unmarshal(c.SortValue, &val); err != nil {
		return "", err
	}
	return val, nil
}

func (c *Cursor) GetTimeValue() (time.Time, error) {
	var val time.Time
	if err := json.Unmarshal(c.SortValue, &val); err != nil {
		return time.Time{}, err
	}
	return val, nil
}

func (c *Cursor) GetIntValue() (int64, error) {
	var val int64
	if err := json.Unmarshal(c.SortValue, &val); err != nil {
		return 0, err
	}
	return val, nil
}
