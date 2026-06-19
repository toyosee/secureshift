package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "runtime"
    "secureshift/internal/server"
    "syscall"
    "time"
)

func main() {
    // Create server instance
    srv := server.NewServer()

    // Start the server in a goroutine
    go func() {
        log.Printf("🚀 SecureShift starting...")
        log.Printf("🌐 Web UI available at: http://localhost:8080")
        log.Printf("📊 Dashboard: http://localhost:8080/dashboard")
        log.Printf("⚡ Press Ctrl+C to stop")
        
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Open browser automatically
    go openBrowser("http://localhost:8080")

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("🛑 Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server shutdown failed: %v", err)
    }
    log.Println("✅ Server stopped gracefully")
}

func openBrowser(url string) {
    // Give the server a moment to start
    time.Sleep(1 * time.Second)
    
    var err error
    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    default:
        err = fmt.Errorf("unsupported platform")
    }
    if err != nil {
        log.Printf("⚠️ Could not open browser automatically: %v", err)
        log.Printf("📍 Please open: %s", url)
    }
}