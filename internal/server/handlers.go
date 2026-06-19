package server

import (
    "archive/zip"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "secureshift/internal/models"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jung-kurt/gofpdf/v2"
)

// Helper function to write JSON responses
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        log.Printf("⚠️ Failed to encode JSON response: %v", err)
    }
}

// Helper function to write error responses
func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, map[string]interface{}{
        "error":   true,
        "message": message,
        "status":  status,
    })
}

func (s *Server) handleUploadScan(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Parse multipart form with 500MB limit
    if err := r.ParseMultipartForm(500 << 20); err != nil {
        writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
        return
    }

    scanID := uuid.New().String()
    tempDir := filepath.Join(os.TempDir(), "secureshift", scanID)
    if err := os.MkdirAll(tempDir, 0755); err != nil {
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create temp directory: %v", err))
        return
    }

    // Get all files from the upload
    files := r.MultipartForm.File["files[]"]
    uploadedCount := 0

    if len(files) == 0 {
        // Try single file upload
        file, header, err := r.FormFile("file")
        if err != nil {
            writeError(w, http.StatusBadRequest, "No files uploaded")
            return
        }
        defer file.Close()

        // Check if it's a zip file
        if strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
            // Save zip file temporarily
            zipPath := filepath.Join(tempDir, header.Filename)
            dst, err := os.Create(zipPath)
            if err != nil {
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save zip: %v", err))
                return
            }

            if _, err := io.Copy(dst, file); err != nil {
                dst.Close()
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to copy zip: %v", err))
                return
            }
            dst.Close()

            // Extract zip
            log.Printf("📦 Extracting zip file: %s", zipPath)
            if err := extractZip(zipPath, tempDir); err != nil {
                log.Printf("⚠️ Failed to extract zip: %v", err)
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to extract zip: %v", err))
                return
            }

            // Remove zip file after extraction
            os.Remove(zipPath)
            uploadedCount = 1
            log.Printf("📁 Extracted zip to %s", tempDir)
        } else {
            // Save single file
            targetPath := filepath.Join(tempDir, header.Filename)
            if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create directory: %v", err))
                return
            }

            dst, err := os.Create(targetPath)
            if err != nil {
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save file: %v", err))
                return
            }
            defer dst.Close()

            if _, err := io.Copy(dst, file); err != nil {
                writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to copy file: %v", err))
                return
            }
            uploadedCount = 1
            log.Printf("📁 Uploaded file: %s", header.Filename)
        }
    } else {
        // Handle multiple files (directory upload from browser)
        for _, fileHeader := range files {
            file, err := fileHeader.Open()
            if err != nil {
                log.Printf("⚠️ Failed to open uploaded file: %v", err)
                continue
            }

            // Get relative path from the filename
            relPath := fileHeader.Filename
            relPath = filepath.Clean(relPath)
            if strings.Contains(relPath, "..") {
                log.Printf("⚠️ Skipping potentially malicious path: %s", relPath)
                file.Close()
                continue
            }

            targetPath := filepath.Join(tempDir, relPath)
            if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
                log.Printf("⚠️ Failed to create directory: %v", err)
                file.Close()
                continue
            }

            dst, err := os.Create(targetPath)
            if err != nil {
                log.Printf("⚠️ Failed to create file: %v", err)
                file.Close()
                continue
            }

            if _, err := io.Copy(dst, file); err != nil {
                log.Printf("⚠️ Failed to copy file: %v", err)
                dst.Close()
                file.Close()
                continue
            }

            dst.Close()
            file.Close()
            uploadedCount++
        }
        log.Printf("📁 Uploaded %d files to %s", uploadedCount, tempDir)
    }

    if uploadedCount == 0 {
        writeError(w, http.StatusBadRequest, "No files were successfully uploaded")
        return
    }

    // Start scan in background
    go s.scanner.ScanProject(scanID, tempDir)

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "scan_id":        scanID,
        "status":         "scanning",
        "message":        fmt.Sprintf("Scan started with %d files", uploadedCount),
        "uploaded_files": uploadedCount,
    })
}

// extractZip extracts a zip file to the target directory
func extractZip(zipPath, targetDir string) error {
    reader, err := zip.OpenReader(zipPath)
    if err != nil {
        return fmt.Errorf("failed to open zip: %v", err)
    }
    defer reader.Close()

    for _, file := range reader.File {
        // Check for zip slip vulnerability
        filePath := filepath.Join(targetDir, file.Name)
        if !strings.HasPrefix(filePath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
            return fmt.Errorf("invalid file path in zip: %s", file.Name)
        }

        if file.FileInfo().IsDir() {
            if err := os.MkdirAll(filePath, 0755); err != nil {
                return fmt.Errorf("failed to create directory: %v", err)
            }
            continue
        }

        if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
            return fmt.Errorf("failed to create directory: %v", err)
        }

        dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
        if err != nil {
            return fmt.Errorf("failed to create file: %v", err)
        }
        defer dstFile.Close()

        srcFile, err := file.Open()
        if err != nil {
            return fmt.Errorf("failed to open zip file: %v", err)
        }
        defer srcFile.Close()

        if _, err := io.Copy(dstFile, srcFile); err != nil {
            return fmt.Errorf("failed to copy file: %v", err)
        }
    }

    return nil
}

func (s *Server) handleGitScan(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    var req struct {
        URL string `json:"url"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
        return
    }

    if req.URL == "" {
        writeError(w, http.StatusBadRequest, "Git URL is required")
        return
    }

    scanID := uuid.New().String()
    tempDir := filepath.Join(os.TempDir(), "secureshift", scanID)
    if err := os.MkdirAll(tempDir, 0755); err != nil {
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create temp directory: %v", err))
        return
    }

    // Clone repository in background
    go s.scanner.CloneAndScan(scanID, req.URL, tempDir)

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "scan_id": scanID,
        "status":  "cloning",
        "message": "Repository cloning started",
    })
}

func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Extract scan ID from URL path
    path := strings.TrimPrefix(r.URL.Path, "/api/scan/")
    scanID := strings.TrimSuffix(path, "/stream")
    scanID = strings.TrimPrefix(scanID, "/")

    if scanID == "" {
        writeError(w, http.StatusBadRequest, "Scan ID required")
        return
    }

    results, err := s.storage.GetScan(scanID)
    if err != nil {
        writeError(w, http.StatusNotFound, fmt.Sprintf("Scan not found: %v", err))
        return
    }

    writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleStreamScan(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Extract scan ID from URL path
    path := strings.TrimPrefix(r.URL.Path, "/api/scan/")
    scanID := strings.TrimSuffix(path, "/stream")
    scanID = strings.TrimPrefix(scanID, "/")

    if scanID == "" {
        writeError(w, http.StatusBadRequest, "Scan ID required")
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, "Streaming not supported")
        return
    }

    progressChan := make(chan models.ProgressUpdate)
    s.scanner.Subscribe(scanID, progressChan)
    defer s.scanner.Unsubscribe(scanID, progressChan)

    for {
        select {
        case <-r.Context().Done():
            return
        case progress := <-progressChan:
            data, err := json.Marshal(progress)
            if err != nil {
                log.Printf("⚠️ Failed to marshal progress: %v", err)
                continue
            }
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()

            if progress.Status == "completed" || progress.Status == "failed" {
                return
            }
        }
    }
}

func (s *Server) handleIgnoreFinding(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Extract finding ID from URL path
    path := strings.TrimPrefix(r.URL.Path, "/api/findings/")
    findingID := strings.TrimSuffix(path, "/ignore")
    findingID = strings.TrimPrefix(findingID, "/")

    if findingID == "" {
        writeError(w, http.StatusBadRequest, "Finding ID required")
        return
    }

    var req struct {
        Reason string `json:"reason"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
        return
    }

    if err := s.storage.IgnoreFinding(findingID, req.Reason); err != nil {
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to ignore finding: %v", err))
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "status":  "success",
        "message": "Finding ignored",
    })
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
    })
}

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    stats, err := s.storage.GetDashboardStats()
    if err != nil {
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get stats: %v", err))
        return
    }

    // Get recent findings for the dashboard
    recentFindings, err := s.storage.GetRecentFindings(5)
    if err == nil {
        stats["recent_findings"] = recentFindings
    }

    writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    history, err := s.storage.GetScanHistory(30)
    if err != nil {
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get history: %v", err))
        return
    }

    // Ensure we always return an array
    if history == nil {
        history = []map[string]interface{}{}
    }

    writeJSON(w, http.StatusOK, history)
}

// handleGetReport returns a formatted report for a scan
func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Extract scan ID from URL path
    path := strings.TrimPrefix(r.URL.Path, "/api/report/")
    scanID := strings.TrimSuffix(path, "/json")
    scanID = strings.TrimPrefix(scanID, "/")
    scanID = strings.TrimSuffix(scanID, "/pdf")

    if scanID == "" {
        writeError(w, http.StatusBadRequest, "Scan ID required")
        return
    }

    results, err := s.storage.GetScan(scanID)
    if err != nil {
        writeError(w, http.StatusNotFound, fmt.Sprintf("Scan not found: %v", err))
        return
    }

    // Check if PDF is requested
    if strings.HasSuffix(r.URL.Path, "/pdf") {
        s.generatePDFReport(w, results)
        return
    }

    // Return JSON report
    writeJSON(w, http.StatusOK, results)
}

// generatePDFReport creates a PDF report for the scan results
func (s *Server) generatePDFReport(w http.ResponseWriter, results *models.ScanResult) {
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.SetMargins(20, 20, 20)
    pdf.AddPage()

    // Header with logo and title
    pdf.SetFont("Arial", "B", 24)
    pdf.SetTextColor(79, 140, 255)
    pdf.Cell(0, 10, "SecureShift Security Report")
    pdf.Ln(10)

    // Divider
    pdf.SetDrawColor(200, 200, 200)
    pdf.Line(20, 40, 190, 40)
    pdf.Ln(10)

    // Project Info
    pdf.SetFont("Arial", "B", 16)
    pdf.SetTextColor(0, 0, 0)
    pdf.Cell(0, 10, fmt.Sprintf("Project: %s", results.ProjectName))
    pdf.Ln(8)
    
    pdf.SetFont("Arial", "", 12)
    pdf.SetTextColor(80, 80, 80)
    pdf.Cell(0, 8, fmt.Sprintf("Scan ID: %s", results.ID))
    pdf.Ln(8)
    pdf.Cell(0, 8, fmt.Sprintf("Scan Date: %s", results.StartTime.Format("January 2, 2006 15:04:05")))
    pdf.Ln(8)
    pdf.Cell(0, 8, fmt.Sprintf("Files Scanned: %d", results.FilesScanned))
    pdf.Ln(8)
    pdf.Cell(0, 8, fmt.Sprintf("Status: %s", strings.ToUpper(results.Status)))
    pdf.Ln(15)

    // Summary Section
    pdf.SetFont("Arial", "B", 18)
    pdf.SetTextColor(79, 140, 255)
    pdf.Cell(0, 10, "Executive Summary")
    pdf.Ln(10)

    // Summary Box
    pdf.SetDrawColor(200, 200, 200)
    pdf.SetFillColor(245, 247, 250)
    pdf.Rect(20, pdf.GetY(), 170, 40, "F")
    
    summary := results.Summary
    pdf.SetFont("Arial", "", 11)
    pdf.SetTextColor(0, 0, 0)
    pdf.SetXY(30, pdf.GetY()+5)
    
    // Create a table for summary
    summaryData := [][]string{
        {"Critical", fmt.Sprintf("%d", summary.Critical), "High", fmt.Sprintf("%d", summary.High)},
        {"Medium", fmt.Sprintf("%d", summary.Medium), "Low", fmt.Sprintf("%d", summary.Low)},
        {"Total", fmt.Sprintf("%d", summary.Total), "", ""},
    }
    
    for _, row := range summaryData {
        pdf.SetX(30)
        // Critical
        if row[0] == "Critical" {
            pdf.SetTextColor(255, 71, 87)
        } else if row[0] == "High" {
            pdf.SetTextColor(255, 107, 53)
        } else if row[0] == "Medium" {
            pdf.SetTextColor(255, 177, 66)
        } else if row[0] == "Low" {
            pdf.SetTextColor(46, 213, 115)
        } else {
            pdf.SetTextColor(0, 0, 0)
        }
        pdf.Cell(30, 8, row[0])
        pdf.SetTextColor(0, 0, 0)
        pdf.Cell(20, 8, row[1])
        pdf.Cell(30, 8, row[2])
        pdf.Cell(20, 8, row[3])
        pdf.Ln(8)
    }
    pdf.Ln(15)

    // Findings Section
    if len(results.Findings) > 0 {
        pdf.SetFont("Arial", "B", 18)
        pdf.SetTextColor(79, 140, 255)
        pdf.Cell(0, 10, "Detailed Findings")
        pdf.Ln(10)

        // Sort findings by severity (critical first)
        severityWeight := map[string]int{
            "critical": 0,
            "high":     1,
            "medium":   2,
            "low":      3,
        }
        
        // Create a copy of findings and sort
        sortedFindings := make([]models.Finding, len(results.Findings))
        copy(sortedFindings, results.Findings)
        
        // Simple bubble sort by severity
        for i := 0; i < len(sortedFindings); i++ {
            for j := i + 1; j < len(sortedFindings); j++ {
                weightI := severityWeight[sortedFindings[i].Severity]
                weightJ := severityWeight[sortedFindings[j].Severity]
                if weightI > weightJ {
                    sortedFindings[i], sortedFindings[j] = sortedFindings[j], sortedFindings[i]
                }
            }
        }
        
        for _, finding := range sortedFindings {
            if finding.Ignored {
                continue
            }
            
            // Check if we need a new page
            if pdf.GetY() > 240 {
                pdf.AddPage()
            }

            // Severity badge
            severityColor := map[string][]int{
                "critical": {255, 71, 87},
                "high":     {255, 107, 53},
                "medium":   {255, 177, 66},
                "low":      {46, 213, 115},
            }
            
            colors := severityColor[finding.Severity]
            if colors == nil {
                colors = []int{128, 128, 128}
            }
            
            pdf.SetFillColor(colors[0], colors[1], colors[2])
            pdf.SetTextColor(255, 255, 255)
            pdf.SetFont("Arial", "B", 9)
            pdf.Rect(20, pdf.GetY(), 25, 6, "F")
            pdf.SetXY(22, pdf.GetY()+1)
            pdf.Cell(20, 5, strings.ToUpper(finding.Severity))
            pdf.Ln(6)

            // Finding title
            pdf.SetTextColor(0, 0, 0)
            pdf.SetFont("Arial", "B", 13)
            pdf.SetX(50)
            pdf.Cell(0, 8, finding.Title)
            pdf.Ln(8)

            // File location
            pdf.SetFont("Arial", "", 10)
            pdf.SetTextColor(80, 80, 80)
            pdf.SetX(50)
            pdf.Cell(0, 6, fmt.Sprintf("File: %s", finding.File))
            if finding.Line > 0 {
                pdf.Cell(0, 6, fmt.Sprintf(" (Line: %d)", finding.Line))
            }
            pdf.Ln(8)

            // Description
            pdf.SetFont("Arial", "", 11)
            pdf.SetTextColor(0, 0, 0)
            pdf.SetX(50)
            pdf.MultiCell(140, 6, finding.Description, "", "", false)
            pdf.Ln(2)

            // Suggestion
            if finding.Suggestion != "" {
                pdf.SetFont("Arial", "I", 10)
                pdf.SetTextColor(46, 213, 115)
                pdf.SetX(50)
                pdf.MultiCell(140, 5, fmt.Sprintf("💡 %s", finding.Suggestion), "", "", false)
            }
            pdf.Ln(8)

            // Divider
            pdf.SetDrawColor(220, 220, 220)
            pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
            pdf.Ln(5)
        }
    } else {
        pdf.SetFont("Arial", "B", 16)
        pdf.SetTextColor(46, 213, 115)
        pdf.Cell(0, 10, "✅ No vulnerabilities found!")
        pdf.Ln(10)
        pdf.SetFont("Arial", "", 12)
        pdf.SetTextColor(80, 80, 80)
        pdf.Cell(0, 8, "Your code passed all security checks. Great job!")
        pdf.Ln(10)
    }

    // Footer
    pdf.SetY(-30)
    pdf.SetDrawColor(200, 200, 200)
    pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
    pdf.Ln(5)
    pdf.SetFont("Arial", "", 8)
    pdf.SetTextColor(128, 128, 128)
    pdf.Cell(0, 5, fmt.Sprintf("Generated by SecureShift v1.0.0 on %s", time.Now().Format("January 2, 2006 15:04:05")))
    pdf.SetX(150)
    pdf.Cell(0, 5, "© 2026 SecureShift")

    // Send PDF to client
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=security-report-%s.pdf", results.ID[:8]))
    
    err := pdf.Output(w)
    if err != nil {
        log.Printf("❌ Failed to generate PDF: %v", err)
        writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate PDF: %v", err))
    }
}