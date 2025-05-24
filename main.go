package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var (
	configFile string // Config file path
	port       int    // Exporter port
)

var root = cobra.Command{
	Use:     "teltonika-exporter [flags] <config_file>",
	Example: "teltonika-exporter --port 15741",
	Short:   "Teltonika Exporter for Prometheus",
	Long:    "A simple exporter for Teltonika devices, exposing metrics to Prometheus.",
	RunE: func(cmd *cobra.Command, args []string) error {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(done)

		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		if configFile == "" {
			return fmt.Errorf("no config file specified")
		}

		config, err := ParseConfig(configFile)
		if err != nil {
			return fmt.Errorf("error parsing config file: %w", err)
		}

		metrics := NewMetrics()
		teltonikaCollector := NewCollector(config, metrics)

		registry := prometheus.NewRegistry()
		registry.MustRegister(teltonikaCollector)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html>
			<head><title>Teltonika exporter</title></head>
			<body>
				<h1>Teltonika exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
		})

		http.Handle("/metrics", promhttp.HandlerFor(registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
				Registry:          registry,
			}),
		)

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Info("listen: %s", "error", err)
			}
		}()

		slog.Info(fmt.Sprintf("Server Started on port %d", port))

		<-done
		slog.Info("Terminate signal received")

		return nil
	},
}

func init() {
	root.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	root.Flags().IntVar(&port, "port", 15741, "Exporter port")
}

func main() {
	if err := root.Execute(); err != nil {
		slog.Error("Error executing command", slog.String("error", err.Error()))
	}
}
