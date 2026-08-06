package lib

import (
	"errors"
	"fmt"
	"time"
)

var errTimestampEmpty = errors.New("timestamp is empty")

// ParseRequiredRFC3339 parses a named required timestamp and reports empty
// input separately from malformed input.
func ParseRequiredRFC3339(name string, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%s: %w", name, errTimestampEmpty)
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

// ParseOptionalRFC3339 returns nil for an absent timestamp and otherwise applies
// the same parsing contract as ParseRequiredRFC3339.
func ParseOptionalRFC3339(name string, value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := ParseRequiredRFC3339(name, value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}
