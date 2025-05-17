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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if configFile == "" {
			return fmt.Errorf("no config file specified")
		}

		config, err := ParseConfig(configFile)
		if err != nil {
			return fmt.Errorf("error parsing config file: %w", err)
		}

		monitor, err := Run(ctx, config)
		if err != nil {
			return fmt.Errorf("error creating collector: %w", err)
		}

		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}

		http.Handle("/metrics", promhttp.HandlerFor(monitor.Registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
				Registry:          monitor.Registry,
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
