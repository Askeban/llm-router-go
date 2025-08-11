package main

import (
    "context"
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    cfg "llm-router-go/internal/config"
    "llm-router-go/internal/server"
)

func main() {
    // Parse command line flags
    configPath := flag.String("config", "config/models.yaml", "path to models YAML file")
    addr := flag.String("addr", ":8080", "HTTP listen address")
    flag.Parse()
    // Determine config file from env if provided
    if env := os.Getenv("ROUTER_CONFIG"); env != "" {
        *configPath = env
    }
    // Load models
    models, err := cfg.LoadModels(*configPath)
    if err != nil {
        log.Fatalf("failed to load models from %s: %v", *configPath, err)
    }
    log.Printf("loaded %d models from %s", len(models), *configPath)
    // Create store and populate
    store := cfg.NewConfigStore()
    store.SetModels(models)
    // Watch for changes
    go func() {
        if err := cfg.WatchConfig(*configPath, store, log.Default()); err != nil {
            log.Printf("configuration watcher error: %v", err)
        }
    }()
    // Create HTTP server
    handler := server.NewRouter(store)
    srv := &http.Server{
        Addr:         *addr,
        Handler:      handler,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
    }
    // Handle graceful shutdown
    go func() {
        log.Printf("starting server on %s", *addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("failed to start server: %v", err)
        }
    }()
    // Wait for interrupt signal to gracefully shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Printf("shutting down server...")
    // Attempt graceful shutdown with timeout
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(timeoutCtx); err != nil {
        log.Printf("server shutdown error: %v", err)
    }
    log.Printf("server exited")
}