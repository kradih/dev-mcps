package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/process"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Process.Enabled {
		log.Fatal("Process server is disabled in configuration")
	}

	server := mcp.NewServer("process-server", "1.0.0")

	procServer := process.NewServer(&cfg.Process)
	procServer.RegisterTools(server)

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
