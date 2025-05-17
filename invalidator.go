package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type WifiClient struct {
	Device string
	Mac    string
	Radio  string
}

func (wc WifiClient) String() string {
	return fmt.Sprintf("%s^%s^%s", wc.Device, wc.Mac, wc.Radio)
}

// Invalidator deletes the metrics for clients that have not been seen for a while.
type Invalidator struct {
	monitor *Monitor
	clients map[string]time.Time
	mtx     sync.Mutex
}

func NewInvalidator(ctx context.Context, monitor *Monitor) *Invalidator {
	wi := &Invalidator{
		monitor: monitor,
		clients: make(map[string]time.Time),
		mtx:     sync.Mutex{},
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				wi.invalidate()
			case <-ctx.Done():
				slog.Info("stopping wifi invalidator")
				return
			}
		}
	}()

	return wi
}

func (i *Invalidator) Refresh(client WifiClient) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.clients[client.String()] = time.Now()
}

func (i *Invalidator) invalidate() {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	for k, t := range i.clients {
		if time.Since(t) > 1*time.Minute {
			parts := strings.Split(k, "^")
			if len(parts) != 3 {
				continue // should not happen
			}
			wc := WifiClient{
				Device: parts[0],
				Mac:    parts[1],
				Radio:  parts[2],
			}

			i.monitor.WirelessClientTxRate.DeleteLabelValues(wc.Device, wc.Mac, wc.Radio)
			i.monitor.WirelessClientRxRate.DeleteLabelValues(wc.Device, wc.Mac, wc.Radio)
			i.monitor.WirelessClientSignal.DeleteLabelValues(wc.Device, wc.Mac, wc.Radio)
			i.monitor.WirelessClientNoise.DeleteLabelValues(wc.Device, wc.Mac, wc.Radio)

			delete(i.clients, wc.String())
		}
	}
}
