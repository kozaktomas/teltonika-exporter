package main

import (
	"encoding/json"
	"testing"
)

func TestOptionalNumber_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantValue float64
		wantValid bool
	}{
		{name: "json number", input: `15`, wantValue: 15, wantValid: true},
		{name: "negative json number", input: `-105`, wantValue: -105, wantValid: true},
		{name: "zero is a valid reading", input: `0`, wantValue: 0, wantValid: true},
		{name: "float", input: `12.5`, wantValue: 12.5, wantValid: true},
		{name: "numeric string", input: `"100"`, wantValue: 100, wantValid: true},
		{name: "negative numeric string", input: `"-77"`, wantValue: -77, wantValid: true},
		{name: "padded numeric string", input: `" 20 "`, wantValue: 20, wantValid: true},
		{name: "not available placeholder", input: `"N/A"`, wantValid: false},
		{name: "empty string", input: `""`, wantValid: false},
		{name: "null", input: `null`, wantValid: false},
		{name: "unexpected type", input: `true`, wantValid: false},
		{name: "unexpected object", input: `{}`, wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got OptionalNumber
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal(%s) unexpected error = %v", tt.input, err)
			}

			if got.Valid != tt.wantValid {
				t.Errorf("Unmarshal(%s) valid = %v, want %v", tt.input, got.Valid, tt.wantValid)
			}

			if got.Valid && got.Value != tt.wantValue {
				t.Errorf("Unmarshal(%s) value = %v, want %v", tt.input, got.Value, tt.wantValue)
			}
		})
	}
}

func TestOptionalNumber_UnmarshalJSON_missingFieldStaysInvalid(t *testing.T) {
	t.Parallel()

	var payload struct {
		Sinr OptionalNumber `json:"sinr"`
	}

	if err := json.Unmarshal([]byte(`{}`), &payload); err != nil {
		t.Fatalf("unexpected error = %v", err)
	}

	if payload.Sinr.Valid {
		t.Errorf("absent field valid = true, want false")
	}
}

func TestOptionalNumber_UnmarshalJSON_invalidJSON(t *testing.T) {
	t.Parallel()

	var got OptionalNumber
	if err := got.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Errorf("UnmarshalJSON on malformed JSON error = nil, want error")
	}
}
