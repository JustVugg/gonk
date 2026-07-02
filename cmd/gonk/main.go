package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JustVugg/gonk/internal/config"
	"github.com/JustVugg/gonk/internal/server"
)

var (
	Version   = "1.2.1"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	var (
		configPath = flag.String("config", "gonk.yaml", "Path to configuration file")
		validate   = flag.Bool("validate", false, "Validate configuration and exit")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		printVersion()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *validate {
		log.Println("✓ Configuration is valid")
		os.Exit(0)
	}

	// Setup logging
	setupLogging(cfg.Logging)

	printBanner()

	// Create and start server
	srv := server.New(cfg)

	// Watch for config changes
	if cfg.Server.HotReload {
		go config.Watch(*configPath, func(newCfg *config.Config) {
			log.Println("Configuration reloaded")
			srv.Reload(newCfg)
		})
	}

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("🚀 GONK v%s starting on %s", Version, cfg.Server.Listen)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("👋 GONK shutdown complete")
}

func printVersion() {
	fmt.Printf("GONK API Gateway v%s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Println("\nFeatures:")
	fmt.Println("  ✓ Authorization (RBAC + Scopes)")
	fmt.Println("  ✓ mTLS Support")
	fmt.Println("  ✓ Load Balancing")
	fmt.Println("  ✓ JWT & API Key Auth")
	fmt.Println("  ✓ Circuit Breaker")
	fmt.Println("  ✓ Rate Limiting")
	fmt.Println("  ✓ Caching")
	fmt.Println("  ✓ WebSocket & gRPC Proxy")
}

func printBanner() {
	banner := `
  ╔═══════════════════════════════════════╗
  ║                                       ║
  ║   ██████╗  ██████╗ ███╗   ██╗██╗  ██╗ ║
  ║  ██╔════╝ ██╔═══██╗████╗  ██║██║ ██╔╝ ║
  ║  ██║  ███╗██║   ██║██╔██╗ ██║█████╔╝  ║
  ║  ██║   ██║██║   ██║██║╚██╗██║██╔═██╗  ║
  ║  ╚██████╔╝╚██████╔╝██║ ╚████║██║  ██╗ ║
  ║   ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝ ║
  ║                                       ║
  ║   Edge-Native API Gateway v1.2        ║
  ║   Authorization • mTLS • Load Balance ║
  ║                                       ║
  ╚═══════════════════════════════════════╝
`
	fmt.Println(banner)
}

func setupLogging(cfg config.LoggingConfig) {
	// Configure logging based on config
	if cfg.Output == "stdout" {
		log.SetOutput(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		}
	}

	if cfg.Format == "json" {
		log.SetFlags(0)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}
