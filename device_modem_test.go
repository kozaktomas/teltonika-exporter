package main

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func num(value float64) OptionalNumber {
	return OptionalNumber{Value: value, Valid: true}
}

func TestFirstReported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		preferred   OptionalNumber
		alternative OptionalNumber
		want        OptionalNumber
	}{
		{name: "preferred wins", preferred: num(-94), alternative: num(-105), want: num(-94)},
		{name: "falls back", preferred: OptionalNumber{}, alternative: num(-105), want: num(-105)},
		{name: "zero counts as reported", preferred: num(0), alternative: num(-105), want: num(0)},
		{name: "both missing", preferred: OptionalNumber{}, alternative: OptionalNumber{}, want: OptionalNumber{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := firstReported(tt.preferred, tt.alternative); got != tt.want {
				t.Errorf("firstReported(%v, %v) = %v, want %v", tt.preferred, tt.alternative, got, tt.want)
			}
		})
	}
}

func TestCellMatchesFrequency(t *testing.T) {
	t.Parallel()

	lte := ModemCell{Earfcn: num(1579), NrArfcn: OptionalNumber{}}
	nr := ModemCell{Earfcn: OptionalNumber{}, NrArfcn: num(635232)}

	tests := []struct {
		name      string
		cell      ModemCell
		frequency float64
		want      bool
	}{
		{name: "lte earfcn matches", cell: lte, frequency: 1579, want: true},
		{name: "lte earfcn differs", cell: lte, frequency: 6200, want: false},
		{name: "nr arfcn matches", cell: nr, frequency: 635232, want: true},
		{name: "nr arfcn differs", cell: nr, frequency: 1579, want: false},
		{name: "cell without any arfcn", cell: ModemCell{}, frequency: 1579, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := cellMatchesFrequency(tt.cell, tt.frequency); got != tt.want {
				t.Errorf("cellMatchesFrequency(%v, %v) = %v, want %v", tt.cell, tt.frequency, got, tt.want)
			}
		})
	}
}

func TestEnrichCarrier(t *testing.T) {
	t.Parallel()

	// Shape observed on a RUTX50 on 5G NSA: the NR leg of ca_signal[] carries no
	// radio quality at all, cell_info[] carries it under nr-arfcn.
	cells := []ModemCell{
		{Earfcn: num(1579), Rsrp: OptionalNumber{}, Rsrq: OptionalNumber{}, Sinr: num(10), Bandwidth: num(20)},
		{NrArfcn: num(635232), Rsrp: num(-105), Rsrq: num(-10), Sinr: num(15), Bandwidth: num(100)},
	}

	tests := []struct {
		name    string
		carrier ModemCarrier
		want    ModemCarrier
	}{
		{
			name:    "nr carrier takes its readings from cell_info",
			carrier: ModemCarrier{Band: "5G N78", Frequency: num(635232), Bandwidth: num(100)},
			want: ModemCarrier{
				Band: "5G N78", Frequency: num(635232), Bandwidth: num(100),
				Rsrp: num(-105), Rsrq: num(-10), Sinr: num(15),
			},
		},
		{
			name: "lte carrier keeps its own readings",
			carrier: ModemCarrier{
				Band: "LTE B3", Primary: true, Frequency: num(1579), Bandwidth: num(20),
				Rsrp: num(-94), Rsrq: num(-17), Sinr: num(10), Rssi: num(-54),
			},
			want: ModemCarrier{
				Band: "LTE B3", Primary: true, Frequency: num(1579), Bandwidth: num(20),
				Rsrp: num(-94), Rsrq: num(-17), Sinr: num(10), Rssi: num(-54),
			},
		},
		{
			name:    "unmatched frequency leaves the carrier untouched",
			carrier: ModemCarrier{Band: "LTE B20", Frequency: num(6200)},
			want:    ModemCarrier{Band: "LTE B20", Frequency: num(6200)},
		},
		{
			name:    "carrier without a frequency cannot be matched",
			carrier: ModemCarrier{Band: "5G N78"},
			want:    ModemCarrier{Band: "5G N78"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := enrichCarrier(tt.carrier, cells); got != tt.want {
				t.Errorf("enrichCarrier(%v) = %v, want %v", tt.carrier, got, tt.want)
			}
		})
	}
}

func TestEnrichCarrier_noCells(t *testing.T) {
	t.Parallel()

	carrier := ModemCarrier{Band: "5G N78", Frequency: num(635232)}
	if got := enrichCarrier(carrier, nil); got != carrier {
		t.Errorf("enrichCarrier(%v, nil) = %v, want %v", carrier, got, carrier)
	}
}

func TestDevice_exportModemCarriers(t *testing.T) {
	t.Parallel()

	const activeHeader = `# HELP teltonika_mobile_carrier_active ` +
		"Aggregated carrier currently in use - always 1, the series disappears once the carrier is gone\n" +
		"# TYPE teltonika_mobile_carrier_active gauge\n"

	tests := []struct {
		name  string
		modem ModemStatus
		want  string
	}{
		{
			name:  "modem without carrier aggregation data exports nothing",
			modem: ModemStatus{ID: "2-1"},
			want:  "",
		},
		{
			name: "carrier without a band is skipped",
			modem: ModemStatus{ID: "2-1", CaSignal: []ModemCarrier{
				{Band: "  ", Rsrp: num(-94)},
			}},
			want: "",
		},
		{
			name: "a duplicate band is exported only once",
			modem: ModemStatus{ID: "2-1", CaSignal: []ModemCarrier{
				{Band: "LTE B3", Primary: true, Rsrp: num(-94)},
				{Band: "LTE B3", Rsrp: num(-70)},
			}},
			want: activeHeader +
				`teltonika_mobile_carrier_active{band="LTE B3",device="RUT007",primary="true",sim="2-1"} 1` + "\n" +
				"# HELP teltonika_mobile_carrier_rsrp RSRP value of a single aggregated carrier in dBm\n" +
				"# TYPE teltonika_mobile_carrier_rsrp gauge\n" +
				`teltonika_mobile_carrier_rsrp{band="LTE B3",device="RUT007",sim="2-1"} -94` + "\n",
		},
		{
			name: "readings the modem did not report are left out, a reported zero is kept",
			modem: ModemStatus{ID: "2-1", CaSignal: []ModemCarrier{
				{Band: "5G N78", Sinr: num(0)},
			}},
			want: activeHeader +
				`teltonika_mobile_carrier_active{band="5G N78",device="RUT007",primary="false",sim="2-1"} 1` + "\n" +
				"# HELP teltonika_mobile_carrier_sinr SINR value of a single aggregated carrier in dB\n" +
				"# TYPE teltonika_mobile_carrier_sinr gauge\n" +
				`teltonika_mobile_carrier_sinr{band="5G N78",device="RUT007",sim="2-1"} 0` + "\n",
		},
		{
			name: "the nr leg is exported with the readings borrowed from cell_info",
			modem: ModemStatus{
				ID:       "2-1",
				CaSignal: []ModemCarrier{{Band: "5G N78", Frequency: num(635232), Bandwidth: num(100)}},
				CellInfo: []ModemCell{{NrArfcn: num(635232), Rsrp: num(-105), Rsrq: num(-10), Sinr: num(15)}},
			},
			want: activeHeader +
				`teltonika_mobile_carrier_active{band="5G N78",device="RUT007",primary="false",sim="2-1"} 1` + "\n" +
				"# HELP teltonika_mobile_carrier_bandwidth_mhz Channel bandwidth of a single aggregated carrier in MHz\n" +
				"# TYPE teltonika_mobile_carrier_bandwidth_mhz gauge\n" +
				`teltonika_mobile_carrier_bandwidth_mhz{band="5G N78",device="RUT007",sim="2-1"} 100` + "\n" +
				"# HELP teltonika_mobile_carrier_rsrp RSRP value of a single aggregated carrier in dBm\n" +
				"# TYPE teltonika_mobile_carrier_rsrp gauge\n" +
				`teltonika_mobile_carrier_rsrp{band="5G N78",device="RUT007",sim="2-1"} -105` + "\n" +
				"# HELP teltonika_mobile_carrier_rsrq RSRQ value of a single aggregated carrier in dB\n" +
				"# TYPE teltonika_mobile_carrier_rsrq gauge\n" +
				`teltonika_mobile_carrier_rsrq{band="5G N78",device="RUT007",sim="2-1"} -10` + "\n" +
				"# HELP teltonika_mobile_carrier_sinr SINR value of a single aggregated carrier in dB\n" +
				"# TYPE teltonika_mobile_carrier_sinr gauge\n" +
				`teltonika_mobile_carrier_sinr{band="5G N78",device="RUT007",sim="2-1"} 15` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			device := Device{name: "RUT007", metrics: NewMetrics()}
			collect := func(ch chan<- prometheus.Metric) {
				device.exportModemCarriers(ch, tt.modem)
			}

			err := testutil.CollectAndCompare(prometheus.CollectorFunc(collect), strings.NewReader(tt.want))
			require.NoError(t, err)
		})
	}
}

func TestDevice_exportModemStatus_optionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		modem         ModemStatus
		wantQuality   bool
		wantTypeLabel string
	}{
		{
			name:          "quality and network type are exported when reported",
			modem:         ModemStatus{ID: "2-1", SignalQuality: num(40), Ntype: "5G-NSA"},
			wantQuality:   true,
			wantTypeLabel: "5G-NSA",
		},
		{
			name:        "an unreported quality is not exported as zero",
			modem:       ModemStatus{ID: "2-1", Ntype: "LTE"},
			wantQuality: false, wantTypeLabel: "LTE",
		},
		{
			name:  "a blank network type is skipped",
			modem: ModemStatus{ID: "2-1", Ntype: "  "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			device := Device{name: "RUT007", metrics: NewMetrics()}
			collect := func(ch chan<- prometheus.Metric) {
				device.exportModemStatus(ch, tt.modem)
			}

			quality := testutil.CollectAndCount(
				prometheus.CollectorFunc(collect), "teltonika_mobile_signal_quality")
			if (quality > 0) != tt.wantQuality {
				t.Errorf("signal_quality series = %d, want reported = %v", quality, tt.wantQuality)
			}

			want := ""
			if tt.wantTypeLabel != "" {
				want = "# HELP teltonika_mobile_network_type " +
					"Mobile network type currently in use - always 1, the type is carried by the type label\n" +
					"# TYPE teltonika_mobile_network_type gauge\n" +
					`teltonika_mobile_network_type{device="RUT007",sim="2-1",type="` + tt.wantTypeLabel + `"} 1` + "\n"
			}

			err := testutil.CollectAndCompare(prometheus.CollectorFunc(collect),
				strings.NewReader(want), "teltonika_mobile_network_type")
			require.NoError(t, err)
		})
	}
}
