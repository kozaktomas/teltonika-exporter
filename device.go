package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	SectionSystem   = "system"
	SectionModem    = "modem"
	SectionWireless = "wireless"
	SectionDhcp     = "dhcp"
)

type Device struct {
	name     string
	schema   string
	host     string
	username string
	password string
	sections []string

	client     *http.Client
	metrics    Metrics
	translator *Translator
	token      string

	ctx context.Context
	mtx sync.Mutex
}

func (d *Device) Collect(ch chan<- prometheus.Metric) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if err := d.authenticate(); err != nil {
		slog.Error("failed to authenticate", "error", err)
		return
	}

	wg := sync.WaitGroup{}
	for _, section := range d.sections {
		switch section {
		case SectionSystem:
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.collectSystemDeviceUsageStatus(ch)
			}()
		case SectionModem:
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.collectModemStatus(ch)
			}()
		case SectionWireless:
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.collectWirelessInterfacesStatus(ch)
			}()
		case SectionDhcp:
			wg.Add(2)
			go func() {
				defer wg.Done()
				d.collectDhcpLeasesIPv4Status(ch)
			}()
			go func() {
				defer wg.Done()
				d.collectDhcpLeasesIPv6Status(ch)
			}()
		}
	}

	wg.Wait()
}

func (d *Device) authenticate() error {
	if d.token != "" {
		valid, _ := d.checkCurrentToken()
		if valid {
			return nil // token is still valid
		}
	}

	url := d.buildUrl("/login")
	requestBody, err := json.Marshal(LoginRequest{
		Username: d.username,
		Password: d.password,
	})
	if err != nil {
		return fmt.Errorf("failed to create auth body: %w", err)
	}

	request, err := http.NewRequestWithContext(d.ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpResponse, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() {
		if err := httpResponse.Body.Close(); err != nil {
			slog.Error("failed to close httpResponse body", "error", err)
		}
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s", httpResponse.Status)
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("failed to read httpResponse body: %w", err)
	}

	var output LoginResponse
	if err := json.Unmarshal(responseBody, &output); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if !output.Success {
		return fmt.Errorf("authentication failed: %s", string(responseBody))
	}

	d.token = output.Data.Token
	return nil
}

// checkCurrentToken checks if the current token is still valid.
// the server prolongs the token if it is still valid.
func (d *Device) checkCurrentToken() (bool, error) {
	var status SessionStatusResponse
	if err := d.get("/session/status", d.token, &status); err != nil {
		slog.Error("failed to get session status", "error", err)
		return false, fmt.Errorf("failed to get session status: %w", err)
	}

	if !status.Success {
		return false, fmt.Errorf("token check failed")
	}

	if !status.Data.Active {
		return false, nil // inactive token
	}

	return true, nil
}

// collectModemStatus fetches /modems/status and exports every metric derived
// from it, for each modem the device reports.
func (d *Device) collectModemStatus(ch chan<- prometheus.Metric) {
	var status ModemStatusResponse
	if err := d.get("/modems/status", d.token, &status); err != nil {
		slog.Error("failed to get modem status", "error", err)
		return
	}

	for _, modem := range status.Data {
		d.exportModemStatus(ch, modem)
		d.exportModemCarriers(ch, modem)
	}
}

// exportModemStatus exports the scalar readings of a single modem. The radio
// values among them describe the primary carrier only; the remaining carriers
// are handled by exportModemCarriers.
func (d *Device) exportModemStatus(ch chan<- prometheus.Metric, modem ModemStatus) {
	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_signal_strength"],
		prometheus.GaugeValue,
		float64(modem.Rssi),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_rsrp"],
		prometheus.GaugeValue,
		float64(modem.Rsrp),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_rsrq"],
		prometheus.GaugeValue,
		float64(modem.Rsrq),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_sinr"],
		prometheus.GaugeValue,
		float64(modem.Sinr),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_data_received"],
		prometheus.GaugeValue,
		float64(modem.Rxbytes),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_data_sent"],
		prometheus.GaugeValue,
		float64(modem.Txbytes),
		d.name, modem.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_temperature"],
		prometheus.GaugeValue,
		float64(modem.Temperature),
		d.name, modem.ID,
	)

	inserted := 0.0
	if strings.EqualFold(modem.Simstate, "inserted") {
		inserted = 1
	}
	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_mobile_connected"],
		prometheus.GaugeValue,
		inserted,
		d.name, modem.ID,
	)

	if modem.SignalQuality.Valid {
		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_signal_quality"],
			prometheus.GaugeValue,
			modem.SignalQuality.Value,
			d.name, modem.ID,
		)
	}

	if ntype := strings.TrimSpace(modem.Ntype); ntype != "" {
		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_network_type"],
			prometheus.GaugeValue,
			1,
			d.name, modem.ID, ntype,
		)
	}
}

// exportModemCarriers exports one set of metrics per aggregated carrier the
// modem currently uses. Carriers come and go, so their presence is exported as
// teltonika_mobile_carrier_active rather than encoded into a single stateful
// series; series of carriers that are no longer aggregated simply stop being
// emitted. Carriers without a band, and duplicate bands - which would produce a
// duplicate label set and make the whole scrape fail - are skipped.
func (d *Device) exportModemCarriers(ch chan<- prometheus.Metric, modem ModemStatus) {
	exported := make(map[string]struct{}, len(modem.CaSignal))

	for _, carrier := range modem.CaSignal {
		band := strings.TrimSpace(carrier.Band)
		if band == "" {
			slog.Debug("skipping carrier without a band", "host", d.host, "modem", modem.ID)
			continue
		}

		if _, duplicate := exported[band]; duplicate {
			slog.Debug("skipping duplicate carrier band", "host", d.host, "modem", modem.ID, "band", band)
			continue
		}
		exported[band] = struct{}{}

		enriched := enrichCarrier(carrier, modem.CellInfo)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_carrier_active"],
			prometheus.GaugeValue,
			1,
			d.name, modem.ID, band, strconv.FormatBool(carrier.Primary),
		)

		d.exportCarrierValue(ch, "teltonika_mobile_carrier_rsrp", modem.ID, band, enriched.Rsrp)
		d.exportCarrierValue(ch, "teltonika_mobile_carrier_rsrq", modem.ID, band, enriched.Rsrq)
		d.exportCarrierValue(ch, "teltonika_mobile_carrier_sinr", modem.ID, band, enriched.Sinr)
		d.exportCarrierValue(ch, "teltonika_mobile_carrier_rssi", modem.ID, band, enriched.Rssi)
		d.exportCarrierValue(ch, "teltonika_mobile_carrier_bandwidth_mhz", modem.ID, band, enriched.Bandwidth)
	}
}

// exportCarrierValue emits a single per-carrier gauge identified by metric, or
// nothing at all when the modem did not report the value. Skipping is
// deliberate: 0 is a legitimate SINR and would be indistinguishable from a
// missing reading.
func (d *Device) exportCarrierValue(
	ch chan<- prometheus.Metric,
	metric, modemID, band string,
	value OptionalNumber,
) {
	if !value.Valid {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		d.metrics[metric],
		prometheus.GaugeValue,
		value.Value,
		d.name, modemID, band,
	)
}

// enrichCarrier fills in the readings that ca_signal[] left out from the
// cell_info[] entry describing the same carrier, matched on its frequency
// (earfcn for LTE, nr-arfcn for NR). On 5G NSA this is what makes the NR leg's
// RSRP, RSRQ and SINR available at all, since ca_signal[] reports only its
// band, frequency and bandwidth. The carrier is returned unchanged when no cell
// matches.
func enrichCarrier(carrier ModemCarrier, cells []ModemCell) ModemCarrier {
	if !carrier.Frequency.Valid {
		return carrier
	}

	for _, cell := range cells {
		if !cellMatchesFrequency(cell, carrier.Frequency.Value) {
			continue
		}

		carrier.Rsrp = firstReported(carrier.Rsrp, cell.Rsrp)
		carrier.Rsrq = firstReported(carrier.Rsrq, cell.Rsrq)
		carrier.Sinr = firstReported(carrier.Sinr, cell.Sinr)
		carrier.Rssi = firstReported(carrier.Rssi, cell.Rssi)
		carrier.Bandwidth = firstReported(carrier.Bandwidth, cell.Bandwidth)

		break
	}

	return carrier
}

// cellMatchesFrequency reports whether cell describes the carrier sitting on
// the given ARFCN. A cell carries either an LTE earfcn or an NR nr-arfcn, never
// both, so the two are compared in turn.
func cellMatchesFrequency(cell ModemCell, frequency float64) bool {
	if cell.Earfcn.Valid && cell.Earfcn.Value == frequency {
		return true
	}

	return cell.NrArfcn.Valid && cell.NrArfcn.Value == frequency
}

// firstReported returns preferred when it holds a value the device reported,
// and falls back to alternative otherwise.
func firstReported(preferred, alternative OptionalNumber) OptionalNumber {
	if preferred.Valid {
		return preferred
	}

	return alternative
}

func (d *Device) collectSystemDeviceUsageStatus(ch chan<- prometheus.Metric) {
	var status SystemDeviceUsageStatusResponse
	if err := d.get("/system/device/usage/status", d.token, &status); err != nil {
		slog.Error("failed to get system device usage status", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_device_uptime"],
		prometheus.GaugeValue,
		float64(status.Data.UptimeSeconds),
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_cpu_usage"],
		prometheus.GaugeValue,
		status.Data.Loadavg,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_load_min_1"],
		prometheus.GaugeValue,
		status.Data.Load.Min1,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_load_min_5"],
		prometheus.GaugeValue,
		status.Data.Load.Min5,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_load_min_15"],
		prometheus.GaugeValue,
		status.Data.Load.Min15,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_ram_total"],
		prometheus.GaugeValue,
		status.Data.Memory.RamTotal*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_ram_used"],
		prometheus.GaugeValue,
		status.Data.Memory.RamUsed*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_ram_free"],
		prometheus.GaugeValue,
		status.Data.Memory.RamFree*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_ram_buffered"],
		prometheus.GaugeValue,
		status.Data.Memory.RamBuffered*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_flash_total"],
		prometheus.GaugeValue,
		status.Data.Memory.FlashTotal*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_flash_used"],
		prometheus.GaugeValue,
		status.Data.Memory.FlashUsed*1e6,
		d.name,
	)

	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_flash_free"],
		prometheus.GaugeValue,
		status.Data.Memory.FlashFree*1e6,
		d.name,
	)
}

func (d *Device) collectDhcpLeasesIPv4Status(ch chan<- prometheus.Metric) {
	var status DhcpLeasesStatusResponse
	if err := d.get("/dhcp/leases/ipv4/status", d.token, &status); err != nil {
		slog.Error("failed to get dhcp leases ipv4 status", "error", err)
		return
	}

	activeLeases := len(status.Data)
	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_dhcp_leases_ipv4"],
		prometheus.GaugeValue,
		float64(activeLeases),
		d.name,
	)
}

func (d *Device) collectDhcpLeasesIPv6Status(ch chan<- prometheus.Metric) {
	var status DhcpLeasesStatusResponse
	if err := d.get("/dhcp/leases/ipv6/status", d.token, &status); err != nil {
		slog.Error("failed to get dhcp leases ipv6 status", "error", err)
		return
	}

	activeLeases := len(status.Data)
	ch <- prometheus.MustNewConstMetric(
		d.metrics["teltonika_dhcp_leases_ipv6"],
		prometheus.GaugeValue,
		float64(activeLeases),
		d.name,
	)
}

func (d *Device) collectWirelessInterfacesStatus(ch chan<- prometheus.Metric) {
	var status WirelessInterfacesStatusResponse
	if err := d.get("/wireless/interfaces/status", d.token, &status); err != nil {
		slog.Error("failed to get wireless interfaces status", "error", err)
		return
	}

	for _, iface := range status.Data {
		if !iface.Up {
			continue // we don't care about down interfaces
		}

		if strings.TrimSpace(iface.Status) != "1" {
			continue // // only for active interfaces
		}

		if iface.Disabled {
			continue // we don't care about disabled interfaces
		}

		for _, device := range iface.Devices {
			ifName := device.IfName
			radio := d.translator.TranslateRadio(device.Name)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_quality"],
				prometheus.GaugeValue,
				float64(device.Quality),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_bitrate"],
				prometheus.GaugeValue,
				float64(device.Bitrate),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_op_class"],
				prometheus.GaugeValue,
				float64(device.OpClass),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_airtime_time_busy"],
				prometheus.CounterValue,
				float64(device.Airtime.TimeBusy),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_airtime_time"],
				prometheus.CounterValue,
				float64(device.Airtime.Time),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_airtime_utilization"],
				prometheus.GaugeValue,
				float64(device.Airtime.Utilization),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_noise"],
				prometheus.GaugeValue,
				float64(device.Noise),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_signal"],
				prometheus.GaugeValue,
				float64(device.Signal),
				d.name, ifName, radio,
			)
		}

		radios := make(map[string]string, len(iface.Clients))
		for _, client := range iface.Clients {
			radios[client.Macaddr] = client.Device
		}

		assoclist, ok := iface.Assoclist.(map[string]interface{})
		if !ok {
			// assoclist might be empty
			slog.Debug("failed to parse assoclist", "host", d.host, "assoclist", iface.Assoclist)
			continue
		}

		for mac, values := range assoclist {
			assoc, ok := values.(map[string]interface{})
			if !ok {
				slog.Error("failed to parse assoclist values", "host", d.host, "assoclist", iface.Assoclist)
				continue
			}

			m := d.translator.TranslateMac(mac)           // translate MAC address
			r := d.translator.TranslateRadio(radios[mac]) // translate radio name

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_client_tx_rate"],
				prometheus.GaugeValue,
				assoc["tx_rate"].(float64), //nolint:forcetypeassert
				d.name, m, r,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_client_rx_rate"],
				prometheus.GaugeValue,
				assoc["rx_rate"].(float64), //nolint:forcetypeassert
				d.name, m, r,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_client_signal"],
				prometheus.GaugeValue,
				assoc["signal"].(float64), //nolint:forcetypeassert
				d.name, m, r,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_client_noise"],
				prometheus.GaugeValue,
				assoc["noise"].(float64), //nolint:forcetypeassert
				d.name, m, r,
			)
		}

	}
}

func (d *Device) get(endpoint, token string, response interface{}) error {
	slog.Debug("Calling API", "url", d.buildUrl(endpoint))

	request, err := http.NewRequestWithContext(d.ctx, http.MethodGet, d.buildUrl(endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "Teltonika Exporter")

	httpResponse, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("token check request failed: %w", err)
	}
	defer func() {
		if err := httpResponse.Body.Close(); err != nil {
			slog.Error("failed to close httpResponse body", "error", err)
		}
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get %s: %s", endpoint, httpResponse.Status)
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("failed to read httpResponse body: %w", err)
	}

	if err := json.Unmarshal(responseBody, response); err != nil {
		fmt.Println(string(responseBody))
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return nil
}

func (d *Device) buildUrl(endpoint string) string {
	return fmt.Sprintf("%s://%s/api%s", d.schema, d.host, endpoint)
}
