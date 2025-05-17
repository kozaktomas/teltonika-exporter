package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	SectionSystem   = "system"
	SectionModem    = "modem"
	SectionWireless = "wireless"
	SectionDhcp     = "dhcp"
)

type Collector struct {
	name     string
	schema   string
	host     string
	username string
	password string
	sections []string

	client      *http.Client
	monitor     *Monitor
	config      *Config
	invalidator *Invalidator

	mtx   Mutex
	token string
}

type DataCollector func()

func Run(ctx context.Context, config *Config) (*Monitor, error) {
	monitor := NewMonitor()                     // one global monitor for all collectors
	invalidator := NewInvalidator(ctx, monitor) // one global invalidator for all collectors

	// create collectors for each device
	for _, device := range config.Devices {
		collector := &Collector{
			name:     device.Name,
			schema:   device.Schema,
			host:     strings.TrimSuffix(device.Host, "/"),
			username: device.Username,
			password: device.Password,
			sections: device.Collect,

			client: &http.Client{
				Timeout: device.Timeout,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, //nolint:gosec
					},
				},
			},
			monitor:     monitor,
			config:      config,
			invalidator: invalidator,

			mtx:   Mutex{},
			token: "",
		}

		// First Collect cycle to check if the device is reachable
		if err := collector.Collect(); err != nil {
			return nil, fmt.Errorf("failed to collect data from %s: %w", device.Host, err)
		}

		// Start collector
		go func(ctx context.Context, collector *Collector) {
			ticker := time.NewTicker(device.ScrapeInternaval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					slog.Info("stopping collector", "host", collector.host)
					return
				case <-ticker.C:
					if err := collector.Collect(); err != nil {
						slog.Error("failed to collect data", "error", err)
					}
				}
			}
		}(ctx, collector)
	}

	return monitor, nil
}

func (c *Collector) Collect() error {
	if c.mtx.IsLocked() {
		// it should never happen, but just in case
		// timeout needs to be lower that collector cycle
		slog.Info("collector is already running")
		return nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := c.authenticate()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	collectors := make([]DataCollector, 0, 5)
	for _, section := range c.sections {
		switch section {
		case SectionSystem:
			collectors = append(collectors, c.collectSystemDeviceUsageStatus)
		case SectionModem:
			collectors = append(collectors, c.collectModemStatus)
		case SectionWireless:
			collectors = append(collectors, c.collectWirelessInterfacesStatus)
		case SectionDhcp:
			collectors = append(collectors, c.collectDhcpLeasesIPv4Status)
			collectors = append(collectors, c.collectDhcpLeasesIPv6Status)
		}
	}

	// run all collectors in parallel
	wg := sync.WaitGroup{}
	wg.Add(len(collectors))
	for _, collector := range collectors {
		go func(collector DataCollector) {
			defer wg.Done()
			collector()
		}(collector)
	}

	wg.Wait()

	return nil
}

func (c *Collector) authenticate() (string, error) {
	if c.token != "" {
		valid, _ := c.checkCurrentToken()
		if valid {
			return c.token, nil
		}
	}

	url := c.buildUrl("/login")
	requestBody, err := json.Marshal(LoginRequest{
		Username: c.username,
		Password: c.password,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create auth body: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpResponse, err := c.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() {
		if err := httpResponse.Body.Close(); err != nil {
			slog.Error("failed to close httpResponse body", "error", err)
		}
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed: %s", httpResponse.Status)
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read httpResponse body: %w", err)
	}

	var output LoginResponse
	if err := json.Unmarshal(responseBody, &output); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if !output.Success {
		return "", fmt.Errorf("authentication failed: %s", string(responseBody))
	}

	c.token = output.Data.Token
	return output.Data.Token, nil
}

func (c *Collector) checkCurrentToken() (bool, error) {
	url := c.buildUrl("/session/status")
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)

	httpResponse, err := c.client.Do(request)
	if err != nil {
		return false, fmt.Errorf("token check request failed: %w", err)
	}
	defer func() {
		if err := httpResponse.Body.Close(); err != nil {
			slog.Error("failed to close httpResponse body", "error", err)
		}
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return false, nil
	}

	type Output struct {
		Success bool `json:"success"`
		Data    struct {
			Active bool `json:"active"`
		} `json:"data"`
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read httpResponse body: %w", err)
	}
	var output Output
	if err := json.Unmarshal(responseBody, &output); err != nil {
		return false, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if !output.Success {
		return false, fmt.Errorf("token check failed: %s", string(responseBody))
	}

	if !output.Data.Active {
		return false, nil
	}

	return true, nil
}

func (c *Collector) collectModemStatus() {
	var status ModemStatusResponse
	if err := c.get("/modems/status", c.token, &status); err != nil {
		slog.Error("failed to get modem status", "error", err)
		return
	}

	for _, sim := range status.Data {
		c.monitor.MobileSignal.WithLabelValues(c.name, sim.ID).Set(float64(sim.Rssi))
		c.monitor.MobileRsrp.WithLabelValues(c.name, sim.ID).Set(float64(sim.Rsrp))
		c.monitor.MobileRsrq.WithLabelValues(c.name, sim.ID).Set(float64(sim.Rsrq))
		c.monitor.MobileSinr.WithLabelValues(c.name, sim.ID).Set(float64(sim.Sinr))
		c.monitor.MobileReceived.WithLabelValues(c.name, sim.ID).Set(float64(sim.Rxbytes))
		c.monitor.MobileSent.WithLabelValues(c.name, sim.ID).Set(float64(sim.Txbytes))
		c.monitor.MobileTemperature.WithLabelValues(c.name, sim.ID).Set(float64(sim.Temperature))

		if strings.EqualFold(sim.Simstate, "inserted") {
			c.monitor.MobileConnected.WithLabelValues(c.name, sim.ID).Set(1)
		} else {
			c.monitor.MobileConnected.WithLabelValues(c.name, sim.ID).Set(0)
		}
	}
}

func (c *Collector) collectSystemDeviceUsageStatus() {
	var status SystemDeviceUsageStatusResponse
	if err := c.get("/system/device/usage/status", c.token, &status); err != nil {
		slog.Error("failed to get system device usage status", "error", err)
		return
	}

	c.monitor.DeviceUptime.WithLabelValues(c.name).Set(float64(status.Data.UptimeSeconds))
	c.monitor.CpuUsage.WithLabelValues(c.name).Set(status.Data.Loadavg)
	c.monitor.LoadMin1.WithLabelValues(c.name).Set(status.Data.Load.Min1)
	c.monitor.LoadMin5.WithLabelValues(c.name).Set(status.Data.Load.Min5)
	c.monitor.LoadMin15.WithLabelValues(c.name).Set(status.Data.Load.Min15)
	// memory is in MB
	c.monitor.RamTotal.WithLabelValues(c.name).Set(status.Data.Memory.RamTotal * 1e6)
	c.monitor.RamUsed.WithLabelValues(c.name).Set(status.Data.Memory.RamUsed * 1e6)
	c.monitor.RamFree.WithLabelValues(c.name).Set(status.Data.Memory.RamFree * 1e6)
	c.monitor.RamBuffered.WithLabelValues(c.name).Set(status.Data.Memory.RamBuffered * 1e6)
	c.monitor.FlashTotal.WithLabelValues(c.name).Set(status.Data.Memory.FlashTotal * 1e6)
	c.monitor.FlashUsed.WithLabelValues(c.name).Set(status.Data.Memory.FlashUsed * 1e6)
	c.monitor.FlashFree.WithLabelValues(c.name).Set(status.Data.Memory.FlashFree * 1e6)
}

func (c *Collector) collectDhcpLeasesIPv4Status() {
	var status DhcpLeasesStatusResponse
	if err := c.get("/dhcp/leases/ipv4/status", c.token, &status); err != nil {
		slog.Error("failed to get dhcp leases ipv4 status", "error", err)
		return
	}

	activeLeases := len(status.Data)
	c.monitor.DhcpLeasesIPv4.WithLabelValues(c.name).Set(float64(activeLeases))
}

func (c *Collector) collectDhcpLeasesIPv6Status() {
	var status DhcpLeasesStatusResponse
	if err := c.get("/dhcp/leases/ipv6/status", c.token, &status); err != nil {
		slog.Error("failed to get dhcp leases ipv6 status", "error", err)
		return
	}

	activeLeases := len(status.Data)
	c.monitor.DhcpLeasesIPv6.WithLabelValues(c.name).Set(float64(activeLeases))
}

func (c *Collector) collectWirelessInterfacesStatus() {
	var status WirelessInterfacesStatusResponse
	if err := c.get("/wireless/interfaces/status", c.token, &status); err != nil {
		slog.Error("failed to get wireless interfaces status", "error", err)
		return
	}

	for _, iface := range status.Data {
		if strings.TrimSpace(iface.Status) != "1" {
			continue // // only for active interfaces
		}

		if iface.Disabled {
			continue // we don't care about disabled interfaces
		}

		for _, device := range iface.Devices {
			ifName := device.IfName
			radio := c.translateRadio(device.Name)

			c.monitor.WirelessDeviceQuality.WithLabelValues(c.name, ifName, radio).Set(float64(device.Quality))
			c.monitor.WirelessDeviceBitrate.WithLabelValues(c.name, ifName, radio).Set(float64(device.Bitrate))
			c.monitor.WirelessDeviceOpClass.WithLabelValues(c.name, ifName, radio).Set(float64(device.OpClass))
			c.monitor.WirelessDeviceAirtimeTimeBusy.WithLabelValues(c.name, ifName, radio).Set(float64(device.Airtime.TimeBusy) / 1e6) // convert microseconds
			c.monitor.WirelessDeviceAirtimeTime.WithLabelValues(c.name, ifName, radio).Set(float64(device.Airtime.Time) / 1e6)         // convert microseconds
			c.monitor.WirelessDeviceAirtimeUtilization.WithLabelValues(c.name, ifName, radio).Set(float64(device.Airtime.Utilization))
			c.monitor.WirelessDeviceNoise.WithLabelValues(c.name, ifName, radio).Set(float64(device.Noise))
			c.monitor.WirelessDeviceSignal.WithLabelValues(c.name, ifName, radio).Set(float64(device.Signal))
		}

		radios := make(map[string]string, len(iface.Clients))
		for _, client := range iface.Clients {
			radios[client.Macaddr] = client.Device
		}

		assoclist, ok := iface.Assoclist.(map[string]interface{})
		if !ok {
			slog.Error("failed to parse assoclist", "host", c.host, "assoclist", iface.Assoclist)
			continue
		}

		for mac, values := range assoclist {
			assoc, ok := values.(map[string]interface{})
			if !ok {
				slog.Error("failed to parse assoclist values", "host", c.host, "assoclist", iface.Assoclist)
				continue
			}

			m := c.translateMac(mac)           // translate MAC address
			r := c.translateRadio(radios[mac]) // translate radio name

			// refresh client in invalidator
			c.invalidator.Refresh(WifiClient{
				Device: c.name,
				Mac:    m,
				Radio:  r,
			})

			c.monitor.WirelessClientTxRate.WithLabelValues(c.name, m, r).Set(assoc["tx_rate"].(float64)) //nolint:forcetypeassert
			c.monitor.WirelessClientRxRate.WithLabelValues(c.name, m, r).Set(assoc["rx_rate"].(float64)) //nolint:forcetypeassert
			c.monitor.WirelessClientSignal.WithLabelValues(c.name, m, r).Set(assoc["signal"].(float64))  //nolint:forcetypeassert
			c.monitor.WirelessClientNoise.WithLabelValues(c.name, m, r).Set(assoc["noise"].(float64))    //nolint:forcetypeassert
		}

	}
}

func (c *Collector) get(endpoint, token string, response interface{}) error {
	request, err := http.NewRequest(http.MethodGet, c.buildUrl(endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)

	httpResponse, err := c.client.Do(request)
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

func (c *Collector) buildUrl(endpoint string) string {
	return fmt.Sprintf("%s://%s/api%s", c.schema, c.host, endpoint)
}

func (c *Collector) translateMac(mac string) string {
	for key, value := range c.config.MacTranslations {
		if strings.EqualFold(mac, key) {
			return value
		}
	}

	return strings.ToUpper(mac)
}

func (c *Collector) translateRadio(radio string) string {
	for key, value := range c.config.RadioTranslations {
		if strings.EqualFold(radio, key) {
			return value
		}
	}

	return radio
}
