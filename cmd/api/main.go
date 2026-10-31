// Package main wires and starts the API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "github.com/plusiv/go-api-boilerplate/internal/adapters/inbound/http"
	"github.com/plusiv/go-api-boilerplate/internal/adapters/outbound/memory"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
	"github.com/plusiv/go-api-boilerplate/internal/infrastructure/config"
	"github.com/plusiv/go-api-boilerplate/internal/infrastructure/profiling"
	"github.com/plusiv/go-api-boilerplate/internal/server"
)

func main() {
	cpuProfile := flag.String("cpuprofile", "", "write a CPU profile to a file")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, *cpuProfile); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, cpuProfile string) (runErr error) {
	if cpuProfile != "" {
		profile, err := profiling.StartCPU(cpuProfile)
		if err != nil {
			return err
		}
		defer func() { runErr = errors.Join(runErr, profile.Close()) }()
	}
	cfg, err := config.Load[config.Config]()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	router, err := httpadapter.NewRouter(usecases.NewItems(memory.NewItems()), logger, httpadapter.RouterConfig{CORSOrigins: cfg.AllowedOrigins, RequestTimeout: cfg.RequestTimeout, PprofEnabled: cfg.Enabled, PprofToken: cfg.Token, MetricsEnabled: cfg.MetricsEnabled, MetricsToken: cfg.MetricsToken})
	if err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return fmt.Errorf("configure router: %w", err)
	}
	srv := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Info("server listening", "address", listener.Addr().String())
	return server.Run(ctx, srv, listener, 10*time.Second)
}
