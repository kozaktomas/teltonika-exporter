package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// OptionalNumber is a numeric field of the Teltonika Web API that is not
// consistently typed. Depending on the field, the firmware and the modem state
// the same value arrives as a JSON number ("sinr": 15), as a string holding a
// number ("bandwidth": "20"), as the placeholder string "N/A", or not at all.
//
// Valid reports whether Value holds a reading the device actually provided.
// It stays false for null, for "N/A" and for anything else that does not parse
// as a number, so a missing reading can be skipped instead of being exported as
// a misleading 0 - which is a legitimate value for SINR and bandwidth.
type OptionalNumber struct {
	Value float64
	Valid bool
}

// UnmarshalJSON decodes a JSON number or a numeric string into n. Values the
// API uses to mean "not reported" - null, "N/A", an empty string, or a value of
// an unexpected type - leave n invalid instead of failing the whole response,
// because a single unreported reading must not cost us every other metric in
// the same payload.
func (n *OptionalNumber) UnmarshalJSON(data []byte) error {
	*n = OptionalNumber{}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal optional number: %w", err)
	}

	switch value := raw.(type) {
	case float64:
		n.Value, n.Valid = value, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			n.Value, n.Valid = parsed, true
		}
	}

	return nil
}
