package main

import (
	"strings"
	"testing"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()

	for name, desc := range metrics {
		if desc == nil {
			t.Errorf("Expected desc for %s to be non-nil", name)
		}

		if strings.HasPrefix(name, "teltonika_") == false {
			t.Errorf("Expected name to start with 'teltonika_'")
		}

		if strings.Contains(desc.String(), name) == false {
			t.Errorf("Expected desc to contain the metric name %s", name)
		}
	}
}
