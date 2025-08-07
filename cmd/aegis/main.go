package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/core"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "", "Log level (debug, info, warn, error, fatal)")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log level if specified
	if *logLevel != "" {
		cfg.Global.LogLevel = *logLevel
	}

	// Initialize logger
	logLevelEnum := logger.ParseLogLevel(cfg.Global.LogLevel)
	logger.SetDefaultLogger(logger.NewLogger(logLevelEnum))

	logger.Info("Starting Aegis-Orchestrator")
	logger.Info("Version: 0.1.0")
	logger.Info("Author: Allen Elzayn")

	// Create the engine first
	engine := core.NewEngine(cfg)

	// Print minimal configuration summary to avoid side effects
	logger.Info("Aegis-Orchestrator Configuration: %s", config.GetConfigSummary(cfg))

	// Initialize the engine
	if err := engine.Initialize(); err != nil {
		logger.Fatal("Failed to initialize engine: %v", err)
	}

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the engine
	if err := engine.Start(ctx); err != nil {
		logger.Fatal("Failed to start engine: %v", err)
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info("Received signal: %v", sig)
	logger.Info("Shutting down...")

	// Stop the engine
	engine.Stop()

	logger.Info("Aegis-Orchestrator stopped")
}
