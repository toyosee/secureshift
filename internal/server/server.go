package server

import (
    "context"
    "io/fs"
    "log"
    "net/http"
    "secureshift/internal/scanner"
    "secureshift/internal/storage"
    "secureshift/webui"
    "strings"
    "time"
)

type Server struct {
    http.Server
    scanner *scanner.Scanner
    storage *storage.Storage
}

func NewServer() *Server {
    // Initialize storage
    store, err := storage.NewStorage("secureshift.db")
    if err != nil {
        log.Fatalf("Failed to initialize storage: %v", err)
    }

    // Initialize scanner
    scan := scanner.NewScanner(store)

    s := &Server{
        scanner: scan,
        storage: store,
    }

    // Setup routes
    mux := http.NewServeMux()
    s.setupRoutes(mux)

    s.Server = http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    return s
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
    // Serve embedded web UI from the webui package
    webFS, err := fs.Sub(webui.WebUI, ".")
    if err != nil {
        log.Printf("⚠️ Failed to load embedded webui: %v", err)
        log.Println("📁 Trying to serve from local ./webui directory...")
        // Fallback: serve from local filesystem (for development)
        mux.Handle("/", http.FileServer(http.Dir("./webui")))
    } else {
        log.Println("✅ Serving web UI from embedded filesystem")
        fileServer := http.FileServer(http.FS(webFS))
        mux.Handle("/", fileServer)
    }

    // API routes - using simple path matching
    mux.HandleFunc("/api/scan/upload", s.handleUploadScan)
    mux.HandleFunc("/api/scan/git", s.handleGitScan)
    mux.HandleFunc("/api/scan/", s.handleScanRoutes)
    mux.HandleFunc("/api/findings/", s.handleFindingRoutes)
    mux.HandleFunc("/api/report/", s.handleReportRoutes)
    mux.HandleFunc("/api/health", s.handleHealth)
    mux.HandleFunc("/api/dashboard/stats", s.handleDashboardStats)
    mux.HandleFunc("/api/dashboard/history", s.handleHistory)

    // Add a catch-all for API routes that might be missing
    mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
        // If no route matched, return 404 as JSON
        writeError(w, http.StatusNotFound, "API endpoint not found")
    })
}

func (s *Server) handleScanRoutes(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/api/scan/")
    parts := strings.Split(path, "/")

    if len(parts) == 0 || parts[0] == "" {
        writeError(w, http.StatusBadRequest, "Scan ID required")
        return
    }

    // Check if it's a stream request
    if len(parts) > 1 && parts[1] == "stream" {
        s.handleStreamScan(w, r)
    } else {
        s.handleGetScan(w, r)
    }
}

func (s *Server) handleFindingRoutes(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/api/findings/")
    parts := strings.Split(path, "/")

    if len(parts) == 0 || parts[0] == "" {
        writeError(w, http.StatusBadRequest, "Finding ID required")
        return
    }

    // Check if it's an ignore request
    if len(parts) > 1 && parts[1] == "ignore" {
        s.handleIgnoreFinding(w, r)
    } else {
        writeError(w, http.StatusNotFound, "Finding endpoint not found")
    }
}

// handleReportRoutes handles report routes
func (s *Server) handleReportRoutes(w http.ResponseWriter, r *http.Request) {
    // Check if it's a PDF request
    if strings.HasSuffix(r.URL.Path, "/pdf") {
        s.handleGetReport(w, r)
        return
    }

    // Otherwise return JSON report
    s.handleGetReport(w, r)
}

func (s *Server) Start() error {
    return s.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    if err := s.storage.Close(); err != nil {
        log.Printf("Warning: error closing storage: %v", err)
    }
    return s.Server.Shutdown(ctx)
}