package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/zrufy/gonk/internal/config"
    "github.com/zrufy/gonk/internal/server"
)

var (
    Version   = "dev"
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
        fmt.Printf("GONK API Gateway\n")
        fmt.Printf("Version:    %s\n", Version)
        fmt.Printf("Build Time: %s\n", BuildTime)
        fmt.Printf("Git Commit: %s\n", GitCommit)
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
