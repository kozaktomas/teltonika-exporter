package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
	mtx        sync.Mutex
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

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
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

func (d *Device) collectModemStatus(ch chan<- prometheus.Metric) {
	var status ModemStatusResponse
	if err := d.get("/modems/status", d.token, &status); err != nil {
		slog.Error("failed to get modem status", "error", err)
		return
	}

	for _, sim := range status.Data {
		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_signal_strength"],
			prometheus.GaugeValue,
			float64(sim.Rssi),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_rsrp"],
			prometheus.GaugeValue,
			float64(sim.Rsrp),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_rsrq"],
			prometheus.GaugeValue,
			float64(sim.Rsrq),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_sinr"],
			prometheus.GaugeValue,
			float64(sim.Sinr),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_data_received"],
			prometheus.GaugeValue,
			float64(sim.Rxbytes),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_data_sent"],
			prometheus.GaugeValue,
			float64(sim.Txbytes),
			d.name, sim.ID,
		)

		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_temperature"],
			prometheus.GaugeValue,
			float64(sim.Temperature),
			d.name, sim.ID,
		)

		inserted := 0.0
		if strings.EqualFold(sim.Simstate, "inserted") {
			inserted = 1
		}
		ch <- prometheus.MustNewConstMetric(
			d.metrics["teltonika_mobile_connected"],
			prometheus.GaugeValue,
			inserted,
			d.name, sim.ID,
		)
	}
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
				prometheus.GaugeValue,
				float64(device.Airtime.TimeBusy),
				d.name, ifName, radio,
			)

			ch <- prometheus.MustNewConstMetric(
				d.metrics["teltonika_wireless_device_airtime_time"],
				prometheus.GaugeValue,
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

	request, err := http.NewRequest(http.MethodGet, d.buildUrl(endpoint), nil)
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
