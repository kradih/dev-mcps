package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/command"
	"github.com/local-mcps/dev-mcps/internal/environment"
	"github.com/local-mcps/dev-mcps/internal/filesystem"
	"github.com/local-mcps/dev-mcps/internal/git"
	"github.com/local-mcps/dev-mcps/internal/process"
	"github.com/local-mcps/dev-mcps/internal/web"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	server := mcp.NewServer("local-mcps-all", "1.0.0")

	if cfg.Filesystem.Enabled {
		fsServer := filesystem.NewServer(&cfg.Filesystem)
		fsServer.RegisterTools(server)
		log.Println("Registered Filesystem tools")
	}

	if cfg.Command.Enabled {
		cmdServer := command.NewServer(&cfg.Command)
		cmdServer.RegisterTools(server)
		log.Println("Registered Command tools")
	}

	if cfg.Environment.Enabled {
		envServer := environment.NewServer(&cfg.Environment)
		envServer.RegisterTools(server)
		log.Println("Registered Environment tools")
	}

	if cfg.Git.Enabled {
		gitServer := git.NewServer(&cfg.Git)
		gitServer.RegisterTools(server)
		log.Println("Registered Git tools")
	}

	if cfg.Process.Enabled {
		procServer := process.NewServer(&cfg.Process)
		procServer.RegisterTools(server)
		log.Println("Registered Process tools")
	}

	if cfg.Web.Enabled {
		webServer := web.NewServer(&cfg.Web)
		webServer.RegisterTools(server)
		log.Println("Registered Web tools")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	log.Println("Starting local-mcps-all server...")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}
}
