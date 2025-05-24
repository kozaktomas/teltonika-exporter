package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevice_Collect(t *testing.T) {
	d := Device{
		name:     "RUT007",
		schema:   "https",
		host:     "localhost",
		username: "root",
		password: "pw",
		sections: []string{SectionModem, SectionDhcp, SectionSystem, SectionWireless},
		client:   mockHttpClient(t),
		metrics:  NewMetrics(),
		translator: &Translator{
			mac: map[string]string{
				"14:25:36:AB:AA:44": "iphone",
			},
			radio: map[string]string{
				"radio0": "wifi_2.4",
			},
		},
		token: "",

		ctx: t.Context(),
		mtx: sync.Mutex{},
	}

	expected, err := os.ReadFile("tests/metrics.txt")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(prometheus.CollectorFunc(d.Collect), bytes.NewReader(expected))
	require.NoError(t, err)
}

type RoundTripperMock struct {
	T *testing.T
}

func (m *RoundTripperMock) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.String(), "/login") {
		content, err := os.ReadFile("tests/login.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/session/status") {
		content, err := os.ReadFile("tests/session_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/modems/status") {
		content, err := os.ReadFile("tests/modems_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/system/device/usage/status") {
		content, err := os.ReadFile("tests/system_device_usage_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/dhcp/leases/ipv4/status") {
		content, err := os.ReadFile("tests/dhcp_leases_ipv4_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/dhcp/leases/ipv6/status") {
		content, err := os.ReadFile("tests/dhcp_leases_ipv6_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	if strings.HasSuffix(req.URL.String(), "/wireless/interfaces/status") {
		content, err := os.ReadFile("tests/wireless_interfaces_status.json")
		assert.NoError(m.T, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}

	m.T.Errorf("unexpected API call - mock is missing")
	return nil, fmt.Errorf("no mock")
}

func mockHttpClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: &RoundTripperMock{T: t},
	}
}
