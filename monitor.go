package main

import "github.com/prometheus/client_golang/prometheus"

// Monitor represents a Prometheus monitor
// It contains Prometheus registry and all available metrics
type Monitor struct {
	Registry *prometheus.Registry

	DeviceUptime *prometheus.GaugeVec
	CpuUsage     *prometheus.GaugeVec
	LoadMin1     *prometheus.GaugeVec
	LoadMin5     *prometheus.GaugeVec
	LoadMin15    *prometheus.GaugeVec
	RamTotal     *prometheus.GaugeVec
	RamUsed      *prometheus.GaugeVec
	RamFree      *prometheus.GaugeVec
	RamBuffered  *prometheus.GaugeVec
	FlashTotal   *prometheus.GaugeVec
	FlashUsed    *prometheus.GaugeVec
	FlashFree    *prometheus.GaugeVec

	DhcpLeasesIPv4 *prometheus.GaugeVec
	DhcpLeasesIPv6 *prometheus.GaugeVec

	MobileConnected   *prometheus.GaugeVec
	MobileSignal      *prometheus.GaugeVec
	MobileSinr        *prometheus.GaugeVec
	MobileRsrp        *prometheus.GaugeVec
	MobileRsrq        *prometheus.GaugeVec
	MobileSent        *prometheus.GaugeVec
	MobileReceived    *prometheus.GaugeVec
	MobileTemperature *prometheus.GaugeVec

	WirelessDeviceQuality            *prometheus.GaugeVec
	WirelessDeviceBitrate            *prometheus.GaugeVec
	WirelessDeviceOpClass            *prometheus.GaugeVec
	WirelessDeviceAirtimeTimeBusy    *prometheus.GaugeVec
	WirelessDeviceAirtimeTime        *prometheus.GaugeVec
	WirelessDeviceAirtimeUtilization *prometheus.GaugeVec
	WirelessDeviceNoise              *prometheus.GaugeVec
	WirelessDeviceSignal             *prometheus.GaugeVec

	WirelessClientTxRate *prometheus.GaugeVec
	WirelessClientRxRate *prometheus.GaugeVec
	WirelessClientSignal *prometheus.GaugeVec
	WirelessClientNoise  *prometheus.GaugeVec
}

// NewMonitor creates a new Monitor
func NewMonitor() *Monitor {
	reg := prometheus.NewRegistry()
	generalLabels := []string{"device"}
	mobileLabels := []string{"device", "sim"}
	wirelessClientLabels := []string{"device", "client", "radio"}
	wirelessDeviceLabels := []string{"device", "interface", "radio"}

	monitor := &Monitor{
		Registry: reg,

		DeviceUptime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_device_uptime",
			Help: "Device uptime",
		}, generalLabels),

		CpuUsage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_cpu_usage",
			Help: "CPU usage over 1 minute",
		}, generalLabels),

		LoadMin1: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_load_min_1",
			Help: "CPU load average over the last minute",
		}, generalLabels),

		LoadMin5: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_load_min_5",
			Help: "CPU load average over the last 5 minutes",
		}, generalLabels),

		LoadMin15: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_load_min_15",
			Help: "CPU load average over the last 15 minutes",
		}, generalLabels),

		//nolint:promlinter
		RamTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_ram_total",
			Help: "Total amount of system memory",
		}, generalLabels),

		RamUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_ram_used",
			Help: "Amount of used system memory",
		}, generalLabels),

		RamFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_ram_free",
			Help: "Amount of free system memory",
		}, generalLabels),

		RamBuffered: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_ram_buffered",
			Help: "Amount of buffered system memory",
		}, generalLabels),

		//nolint:promlinter
		FlashTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_flash_total",
			Help: "Total amount of flash memory",
		}, generalLabels),

		FlashUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_flash_used",
			Help: "Amount of used flash memory",
		}, generalLabels),

		FlashFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_flash_free",
			Help: "Amount of free flash memory",
		}, generalLabels),

		DhcpLeasesIPv4: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_dhcp_leases_ipv4",
			Help: "Count of active DHCP IPv4 leases",
		}, generalLabels),

		DhcpLeasesIPv6: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_dhcp_leases_ipv6",
			Help: "Count of active DHCP IPv6 leases",
		}, generalLabels),

		MobileConnected: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_connected",
			Help: "Mobile network connected 1/0",
		}, mobileLabels),

		MobileSignal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_signal_strength",
			Help: "Mobile signal strength",
		}, mobileLabels),

		MobileSinr: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_sinr",
			Help: "SINR value in dB",
		}, mobileLabels),

		MobileRsrp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_rsrp",
			Help: "RSRP value in dBm",
		}, mobileLabels),

		MobileRsrq: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_rsrq",
			Help: "RSRQ value in dB",
		}, mobileLabels),

		MobileSent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_data_sent",
			Help: "Sent data in bytes",
		}, mobileLabels),

		MobileReceived: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_data_received",
			Help: "Received data in bytes",
		}, mobileLabels),

		MobileTemperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_mobile_temperature",
			Help: "Modem temperature in Celsius",
		}, mobileLabels),

		WirelessDeviceQuality: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_quality",
			Help: "Wireless device quality",
		}, wirelessDeviceLabels),

		WirelessDeviceBitrate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_bitrate",
			Help: "Wireless device bitrate",
		}, wirelessDeviceLabels),

		WirelessDeviceOpClass: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_op_class",
			Help: "Wireless device operating class",
		}, wirelessDeviceLabels),

		WirelessDeviceAirtimeTimeBusy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_airtime_time_busy",
			Help: "Duration of busy airtime for the wireless device",
		}, wirelessDeviceLabels),

		WirelessDeviceAirtimeTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_airtime_time",
			Help: "Total airtime duration for the wireless device",
		}, wirelessDeviceLabels),

		WirelessDeviceAirtimeUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_airtime_utilization",
			Help: "Percentage of time the wireless device is actively transmitting or receiving data",
		}, wirelessDeviceLabels),

		WirelessDeviceNoise: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_noise",
			Help: "Wireless device noise level in dBm",
		}, wirelessDeviceLabels),

		WirelessDeviceSignal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_device_signal",
			Help: "Wireless device signal strength in dBm",
		}, wirelessDeviceLabels),

		WirelessClientTxRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_client_tx_rate",
			Help: "Wireless client transmit rate in bps",
		}, wirelessClientLabels),

		WirelessClientRxRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_client_rx_rate",
			Help: "Wireless client receive rate in bps",
		}, wirelessClientLabels),

		WirelessClientSignal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_client_signal",
			Help: "Wireless client signal strength in dBm",
		}, wirelessClientLabels),

		WirelessClientNoise: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "teltonika_wireless_client_noise",
			Help: "Wireless client noise level in dBm",
		}, wirelessClientLabels),
	}

	reg.MustRegister(
		monitor.DeviceUptime,
		monitor.CpuUsage,
		monitor.LoadMin1,
		monitor.LoadMin5,
		monitor.LoadMin15,
		monitor.RamTotal,
		monitor.RamUsed,
		monitor.RamFree,
		monitor.RamBuffered,
		monitor.FlashTotal,
		monitor.FlashUsed,
		monitor.FlashFree,

		monitor.DhcpLeasesIPv4,
		monitor.DhcpLeasesIPv6,

		monitor.MobileConnected,
		monitor.MobileSignal,
		monitor.MobileSinr,
		monitor.MobileRsrp,
		monitor.MobileRsrq,
		monitor.MobileSent,
		monitor.MobileReceived,
		monitor.MobileTemperature,

		monitor.WirelessDeviceQuality,
		monitor.WirelessDeviceBitrate,
		monitor.WirelessDeviceOpClass,
		monitor.WirelessDeviceAirtimeTimeBusy,
		monitor.WirelessDeviceAirtimeTime,
		monitor.WirelessDeviceAirtimeUtilization,
		monitor.WirelessDeviceNoise,
		monitor.WirelessDeviceSignal,

		monitor.WirelessClientTxRate,
		monitor.WirelessClientRxRate,
		monitor.WirelessClientSignal,
		monitor.WirelessClientNoise,
	)

	return monitor
}
