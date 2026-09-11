package main

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

type SessionStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Active bool `json:"active"`
	} `json:"data"`
}

type ModemStatusResponse struct {
	Success bool          `json:"success"`
	Data    []ModemStatus `json:"data"`
}

// ModemStatus is one modem entry of /modems/status. The scalar signal readings
// (Rsrp, Rsrq, Sinr, Rssi) describe the primary carrier only - on a 5G NSA
// connection that is the LTE anchor, not the NR leg, whose readings live in
// CaSignal and CellInfo.
type ModemStatus struct {
	Sinr          int            `json:"sinr"`
	Temperature   int            `json:"temperature"`
	Simstate      string         `json:"simstate"`
	Txbytes       int            `json:"txbytes"`
	Rsrp          int            `json:"rsrp"`
	Rxbytes       int64          `json:"rxbytes"`
	Rssi          int            `json:"rssi"`
	Rsrq          int            `json:"rsrq"`
	ID            string         `json:"id"`
	Band          string         `json:"band"`
	Conntype      string         `json:"conntype"`
	Ntype         string         `json:"ntype"`
	Netstate      string         `json:"netstate"`
	SignalQuality OptionalNumber `json:"signal_quality"`
	CaSignal      []ModemCarrier `json:"ca_signal"`
	CellInfo      []ModemCell    `json:"cell_info"`
}

// ModemCarrier is a single aggregated carrier from ca_signal[]. An NR carrier
// of a 5G NSA connection typically reports only Band, Frequency and Bandwidth;
// its radio quality has to be taken from the matching ModemCell.
type ModemCarrier struct {
	Band      string         `json:"band"`
	Primary   bool           `json:"primary"`
	Frequency OptionalNumber `json:"frequency"`
	Bandwidth OptionalNumber `json:"bandwidth"`
	Rsrp      OptionalNumber `json:"rsrp"`
	Rsrq      OptionalNumber `json:"rsrq"`
	Sinr      OptionalNumber `json:"sinr"`
	Rssi      OptionalNumber `json:"rssi"`
}

// ModemCell is a single cell from cell_info[]. Exactly one of Earfcn (LTE) and
// NrArfcn (NR) holds a number, the other one is reported as "N/A"; that number
// is what ties the cell to a ModemCarrier of the same frequency.
type ModemCell struct {
	Earfcn    OptionalNumber `json:"earfcn"`
	NrArfcn   OptionalNumber `json:"nr-arfcn"`
	Bandwidth OptionalNumber `json:"bandwidth"`
	Rsrp      OptionalNumber `json:"rsrp"`
	Rsrq      OptionalNumber `json:"rsrq"`
	Sinr      OptionalNumber `json:"sinr"`
	Rssi      OptionalNumber `json:"rssi"`
}

type SystemDeviceUsageStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Memory struct {
			RamBuffered     float64 `json:"ram_buffered"`
			RamTotal        float64 `json:"ram_total"`
			RamUsed         float64 `json:"ram_used"`
			FlashTotal      float64 `json:"flash_total"`
			RamFree         float64 `json:"ram_free"`
			FlashFree       float64 `json:"flash_free"`
			FlashPercentage float64 `json:"flash_percentage"`
			FlashUsed       float64 `json:"flash_used"`
			RamPercentage   float64 `json:"ram_percentage"`
			RamShared       float64 `json:"ram_shared"`
		} `json:"memory"`
		Uptime    string  `json:"uptime"`
		Loadavg   float64 `json:"loadavg"`
		Localtime int64   `json:"localtime"`
		Load      struct {
			Min5  float64 `json:"min5"`
			Min15 float64 `json:"min15"`
			Min1  float64 `json:"min1"`
		} `json:"load"`
		UptimeSeconds int64 `json:"uptime_seconds"`
	} `json:"data"`
}

type DhcpLeasesStatusResponse struct {
	Success bool          `json:"success"`
	Data    []interface{} `json:"data"`
}

type WirelessInterfacesStatusResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		Disabled bool   `json:"disabled"`
		Status   string `json:"status"`
		Up       bool   `json:"up"`
		Devices  []struct {
			IfName         string `json:"ifname"`
			Device         string `json:"device"`
			Pending        bool   `json:"pending"`
			Name           string `json:"name"`
			Up             bool   `json:"up"`
			BeaconInterval int    `json:"beacon_interval"`
			Bssid          string `json:"bssid"`
			BssColor       int    `json:"bss_color"`
			Rrm            struct {
				NeighborReportTx int `json:"neighbor_report_tx"`
			} `json:"rrm"`
			Bitrate int `json:"bitrate"`
			Quality int `json:"quality"`
			OpClass int `json:"op_class"`
			Airtime struct {
				TimeBusy    int `json:"time_busy"`
				Time        int `json:"time"`
				Utilization int `json:"utilization"`
			} `json:"airtime"`
			Wnm struct {
				BssTransitionRequestTx  int `json:"bss_transition_request_tx"`
				BssTransitionResponseRx int `json:"bss_transition_response_rx"`
				BssTransitionQueryRx    int `json:"bss_transition_query_rx"`
			} `json:"wnm"`
			Noise  int `json:"noise"`
			Signal int `json:"signal"`
		} `json:"devices"`
		Assoclist interface{} // https://community.teltonika.lt/t/api-bug-report-empty-assoclist-object/13774
		// Assoclist map[string]struct {
		// 	Noise  int `json:"noise"`
		// 	TxRate int `json:"tx_rate"`
		// 	RxRate int `json:"rx_rate"`
		// 	Signal int `json:"signal"`
		// } `json:"assoclist"`
		Clients []struct {
			TxRate   int    `json:"tx_rate"`
			Device   string `json:"device"`
			Ipaddr   string `json:"ipaddr"`
			Band     string `json:"band"`
			Standard string `json:"standard"`
			Macaddr  string `json:"macaddr"`
			RxRate   int    `json:"rx_rate"`
			Signal   string `json:"signal"`
		} `json:"clients"`
	} `json:"data"`
}
