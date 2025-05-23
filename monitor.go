package main

import "github.com/prometheus/client_golang/prometheus"

type Metrics map[string]*prometheus.Desc

func NewMetrics() Metrics {
	generalLabels := []string{"device"}
	mobileLabels := []string{"device", "sim"}
	wirelessClientLabels := []string{"device", "client", "radio"}
	wirelessDeviceLabels := []string{"device", "interface", "radio"}

	return map[string]*prometheus.Desc{
		"teltonika_device_uptime": prometheus.NewDesc(
			"teltonika_device_uptime",
			"Device uptime",
			generalLabels,
			nil,
		),

		"teltonika_cpu_usage": prometheus.NewDesc(
			"teltonika_cpu_usage",
			"CPU usage over 1 minute",
			generalLabels,
			nil,
		),

		"teltonika_load_min_1": prometheus.NewDesc(
			"teltonika_load_min_1",
			"CPU load average over the last minute",
			generalLabels,
			nil,
		),

		"teltonika_load_min_5": prometheus.NewDesc(
			"teltonika_load_min_5",
			"CPU load average over the last 5 minutes",
			generalLabels,
			nil,
		),

		"teltonika_load_min_15": prometheus.NewDesc(
			"teltonika_load_min_15",
			"CPU load average over the last 15 minutes",
			generalLabels,
			nil,
		),

		//nolint:promlinter
		"teltonika_ram_total": prometheus.NewDesc(
			"teltonika_ram_total",
			"Total amount of system memory",
			generalLabels,
			nil,
		),

		"teltonika_ram_used": prometheus.NewDesc(
			"teltonika_ram_used",
			"Amount of used system memory",
			generalLabels,
			nil,
		),

		"teltonika_ram_free": prometheus.NewDesc(
			"teltonika_ram_free",
			"Amount of free system memory",
			generalLabels,
			nil,
		),

		"teltonika_ram_buffered": prometheus.NewDesc(
			"teltonika_ram_buffered",
			"Amount of buffered system memory",
			generalLabels,
			nil,
		),

		//nolint:promlinter
		"teltonika_flash_total": prometheus.NewDesc(
			"teltonika_flash_total",
			"Total amount of flash memory",
			generalLabels,
			nil,
		),

		"teltonika_flash_used": prometheus.NewDesc(
			"teltonika_flash_used",
			"Amount of used flash memory",
			generalLabels,
			nil,
		),

		"teltonika_flash_free": prometheus.NewDesc(
			"teltonika_flash_free",
			"Amount of free flash memory",
			generalLabels,
			nil,
		),

		"teltonika_dhcp_leases_ipv4": prometheus.NewDesc(
			"teltonika_dhcp_leases_ipv4",
			"Count of active DHCP IPv4 leases",
			generalLabels,
			nil,
		),

		"teltonika_dhcp_leases_ipv6": prometheus.NewDesc(
			"teltonika_dhcp_leases_ipv6",
			"Count of active DHCP IPv6 leases",
			generalLabels,
			nil,
		),

		"teltonika_mobile_connected": prometheus.NewDesc(
			"teltonika_mobile_connected",
			"Mobile network connected 1/0",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_signal_strength": prometheus.NewDesc(
			"teltonika_mobile_signal_strength",
			"Mobile signal strength",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_sinr": prometheus.NewDesc(
			"teltonika_mobile_sinr",
			"SINR value in dB",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_rsrp": prometheus.NewDesc(
			"teltonika_mobile_rsrp",
			"RSRP value in dBm",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_rsrq": prometheus.NewDesc(
			"teltonika_mobile_rsrq",
			"RSRQ value in dB",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_data_sent": prometheus.NewDesc(
			"teltonika_mobile_data_sent",
			"Sent data in bytes",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_data_received": prometheus.NewDesc(
			"teltonika_mobile_data_received",
			"Received data in bytes",
			mobileLabels,
			nil,
		),

		"teltonika_mobile_temperature": prometheus.NewDesc(
			"teltonika_mobile_temperature",
			"Modem temperature in Celsius",
			mobileLabels,
			nil,
		),

		"teltonika_wireless_device_quality": prometheus.NewDesc(
			"teltonika_wireless_device_quality",
			"Wireless device quality",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_bitrate": prometheus.NewDesc(
			"teltonika_wireless_device_bitrate",
			"Wireless device bitrate",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_op_class": prometheus.NewDesc(
			"teltonika_wireless_device_op_class",
			"Wireless device operating class",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_airtime_time_busy": prometheus.NewDesc(
			"teltonika_wireless_device_airtime_time_busy",
			"Duration of busy airtime for the wireless device",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_airtime_time": prometheus.NewDesc(
			"teltonika_wireless_device_airtime_time",
			"Total airtime duration for the wireless device",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_airtime_utilization": prometheus.NewDesc(
			"teltonika_wireless_device_airtime_utilization",
			"Percentage of time the wireless device is actively transmitting or receiving data",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_noise": prometheus.NewDesc(
			"teltonika_wireless_device_noise",
			"Wireless device noise level in dBm",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_device_signal": prometheus.NewDesc(
			"teltonika_wireless_device_signal",
			"Wireless device signal strength in dBm",
			wirelessDeviceLabels,
			nil,
		),

		"teltonika_wireless_client_tx_rate": prometheus.NewDesc(
			"teltonika_wireless_client_tx_rate",
			"Wireless client transmit rate in bps",
			wirelessClientLabels,
			nil,
		),

		"teltonika_wireless_client_rx_rate": prometheus.NewDesc(
			"teltonika_wireless_client_rx_rate",
			"Wireless client receive rate in bps",
			wirelessClientLabels,
			nil,
		),

		"teltonika_wireless_client_signal": prometheus.NewDesc(
			"teltonika_wireless_client_signal",
			"Wireless client signal strength in dBm",
			wirelessClientLabels,
			nil,
		),

		"teltonika_wireless_client_noise": prometheus.NewDesc(
			"teltonika_wireless_client_noise",
			"Wireless client noise level in dBm",
			wirelessClientLabels,
			nil,
		),
	}
}
