package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/git"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Git.Enabled {
		log.Fatal("Git server is disabled in configuration")
	}

	server := mcp.NewServer("git-server", "1.0.0")

	gitServer := git.NewServer(&cfg.Git)
	gitServer.RegisterTools(server)

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
