package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/environment"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Environment.Enabled {
		log.Fatal("Environment server is disabled in configuration")
	}

	server := mcp.NewServer("environment-server", "1.0.0")

	envServer := environment.NewServer(&cfg.Environment)
	envServer.RegisterTools(server)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}
}
