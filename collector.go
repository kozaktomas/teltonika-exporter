package main

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	devices []*Device
}

func NewCollector(config *Config, metrics Metrics) *Collector {
	devices := make([]*Device, len(config.Devices))

	translator := &Translator{
		mac:   config.MacTranslations,
		radio: config.RadioTranslations,
	}

	for i, device := range config.Devices {
		devices[i] = &Device{
			name:     device.Name,
			schema:   device.Schema,
			host:     device.Host,
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

			metrics:    metrics,
			translator: translator,
			token:      "",
		}
	}

	return &Collector{
		devices: devices,
	}
}

func (cc *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

func (cc *Collector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(cc.devices))

	for _, device := range cc.devices {
		go func() {
			defer wg.Done()
			device.Collect(ch)
		}()
	}

	wg.Wait()
}
