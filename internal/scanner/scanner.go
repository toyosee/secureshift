package scanner

import (
    "fmt"
    "log"
	"strings"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "secureshift/internal/models"
    "secureshift/internal/storage"
    "sync"
    "time"
)

type Scanner struct {
    storage  *storage.Storage
    progress map[string]chan models.ProgressUpdate
    mu       sync.RWMutex
}

func NewScanner(store *storage.Storage) *Scanner {
    return &Scanner{
        storage:  store,
        progress: make(map[string]chan models.ProgressUpdate),
    }
}

func (s *Scanner) Subscribe(scanID string, ch chan models.ProgressUpdate) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.progress[scanID] = ch
}

func (s *Scanner) Unsubscribe(scanID string, ch chan models.ProgressUpdate) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.progress, scanID)
}

func (s *Scanner) updateProgress(scanID string, status string, progress int, message string) {
    s.mu.RLock()
    ch, exists := s.progress[scanID]
    s.mu.RUnlock()

    if exists {
        select {
        case ch <- models.ProgressUpdate{
            ScanID:   scanID,
            Status:   status,
            Progress: progress,
            Message:  message,
        }:
        default:
        }
    }
}

func (s *Scanner) ScanProject(scanID string, path string) {
    log.Printf("📂 Starting scan for %s at path: %s", scanID, path)
    s.updateProgress(scanID, "scanning", 0, "Starting scan...")

    // Check if path exists
    if _, err := os.Stat(path); os.IsNotExist(err) {
        log.Printf("❌ Path does not exist: %s", path)
        s.updateProgress(scanID, "failed", 0, fmt.Sprintf("Path does not exist: %s", path))
        return
    }

    // Get all files
    var files []string
    err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
        if err != nil {
            log.Printf("⚠️ Error accessing %s: %v", filePath, err)
            return nil // Skip files we can't access
        }
        if !info.IsDir() {
            // Skip binary files, images, etc.
            ext := filepath.Ext(filePath)
            supportedExts := map[string]bool{
                ".go": true, ".js": true, ".py": true, ".java": true,
                ".json": true, ".yaml": true, ".yml": true, ".toml": true,
                ".txt": true, ".md": true, ".sh": true, ".ts": true,
                ".jsx": true, ".tsx": true, ".rb": true, ".php": true,
                ".c": true, ".cpp": true, ".h": true, ".hpp": true,
                ".rs": true, ".swift": true, ".kt": true, ".cs": true,
                ".html": true, ".css": true, ".xml": true,
                ".vue": true, ".svelte": true, ".scss": true, ".less": true,
                ".sql": true, ".ps1": true, ".bat": true,
                ".tf": true, ".tfvars": true, ".hcl": true, // Terraform
                ".dockerfile": true, ".makefile": true, ".cmake": true,
            }
            if supportedExts[ext] {
                files = append(files, filePath)
            }
        }
        return nil
    })

    if err != nil {
        log.Printf("❌ Error walking path: %v", err)
        s.updateProgress(scanID, "failed", 0, fmt.Sprintf("Error walking path: %v", err))
        return
    }

    if len(files) == 0 {
        log.Printf("⚠️ No supported files found in %s", path)
        s.updateProgress(scanID, "failed", 0, "No supported files found to scan")
        return
    }

    log.Printf("📊 Found %d files to scan", len(files))
    s.updateProgress(scanID, "scanning", 10, fmt.Sprintf("Found %d files to scan", len(files)))

    // Initialize results
    result := &models.ScanResult{
        ID:           scanID,
        ProjectName:  filepath.Base(path),
        Status:       "scanning",
        StartTime:    time.Now(),
        Findings:     []models.Finding{},
        FilesScanned: 0,
    }

    // Scan for secrets
    log.Printf("🔍 Scanning for secrets in %d files...", len(files))
    s.updateProgress(scanID, "scanning", 30, "Scanning for secrets...")
    secretFindings := s.scanSecrets(files)
    result.Findings = append(result.Findings, secretFindings...)
    log.Printf("🔍 Found %d secret issues", len(secretFindings))

    // Scan for dependencies
    log.Printf("📦 Checking dependencies...")
    s.updateProgress(scanID, "scanning", 60, "Checking dependencies...")
    depFindings := s.scanDependencies(path)
    result.Findings = append(result.Findings, depFindings...)
    log.Printf("📦 Found %d dependency issues", len(depFindings))

    // Scan for code vulnerabilities (basic)
    log.Printf("💻 Analyzing code patterns...")
    s.updateProgress(scanID, "scanning", 85, "Analyzing code patterns...")
    codeFindings := s.scanCodePatterns(files)
    result.Findings = append(result.Findings, codeFindings...)
    log.Printf("💻 Found %d code pattern issues", len(codeFindings))

    // Calculate summary
    result.Summary = calculateSummary(result.Findings)
    result.FilesScanned = len(files)
    result.EndTime = time.Now()
    result.Status = "completed"

    // Store results
    log.Printf("💾 Saving scan results for %s...", scanID)
    if err := s.storage.SaveScan(result); err != nil {
        log.Printf("❌ Error saving scan results: %v", err)
        s.updateProgress(scanID, "failed", 0, fmt.Sprintf("Error saving results: %v", err))
        return
    }

    log.Printf("✅ Scan complete! Found %d issues", result.Summary.Total)
    s.updateProgress(scanID, "completed", 100, fmt.Sprintf("Scan complete! Found %d issues", result.Summary.Total))
}

func (s *Scanner) CloneAndScan(scanID string, repoURL string, path string) {
    log.Printf("📦 Starting clone for %s from %s", scanID, repoURL)
    s.updateProgress(scanID, "cloning", 0, fmt.Sprintf("Cloning repository: %s", repoURL))
    
    // Try to use git command line first (more reliable)
    err := s.cloneWithGit(repoURL, path)
    if err != nil {
        log.Printf("⚠️ Git clone failed: %v", err)
        s.updateProgress(scanID, "failed", 0, fmt.Sprintf("Failed to clone repository: %v", err))
        return
    }
    
    s.updateProgress(scanID, "cloning", 100, "Repository cloned successfully")
    log.Printf("✅ Repository cloned to %s", path)
    
    // Proceed with scan
    s.ScanProject(scanID, path)
}

// cloneWithGit uses the git command line to clone (cross-platform)
func (s *Scanner) cloneWithGit(repoURL, path string) error {
    // Determine git command based on OS
    gitCmd := "git"
    if runtime.GOOS == "windows" {
        gitCmd = "git.exe"
    }
    
    // Check if git is available by executing git --version
    checkCmd := exec.Command(gitCmd, "--version")
    if err := checkCmd.Run(); err != nil {
        // Try common git paths as fallback
        if runtime.GOOS == "windows" {
            // Common Windows git paths
            commonPaths := []string{
                "C:\\Program Files\\Git\\bin\\git.exe",
                "C:\\Program Files (x86)\\Git\\bin\\git.exe",
                "C:\\Users\\" + os.Getenv("USERNAME") + "\\AppData\\Local\\Programs\\Git\\bin\\git.exe",
            }
            for _, gitPath := range commonPaths {
                if _, err := os.Stat(gitPath); err == nil {
                    gitCmd = gitPath
                    break
                }
            }
            // If still not found, check if git is in PATH via where command
            if gitCmd == "git.exe" {
                whereCmd := exec.Command("where", "git")
                if output, err := whereCmd.Output(); err == nil && len(output) > 0 {
                    gitCmd = strings.TrimSpace(string(output))
                }
            }
        } else if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
            // Try common Unix git paths
            commonPaths := []string{
                "/usr/bin/git",
                "/usr/local/bin/git",
                "/opt/homebrew/bin/git",
            }
            for _, gitPath := range commonPaths {
                if _, err := os.Stat(gitPath); err == nil {
                    gitCmd = gitPath
                    break
                }
            }
        }
        
        // Final check
        checkCmd = exec.Command(gitCmd, "--version")
        if err := checkCmd.Run(); err != nil {
            return fmt.Errorf("git not found: please install git and ensure it's in your PATH")
        }
    }
    
    log.Printf("🔧 Using git command: %s", gitCmd)
    
    // Use git command to clone
    cmd := exec.Command(gitCmd, "clone", "--depth", "1", repoURL, path)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git clone failed: %v - %s", err, string(output))
    }
    
    return nil
}

func calculateSummary(findings []models.Finding) models.Summary {
    summary := models.Summary{}
    for _, f := range findings {
        if f.Ignored {
            continue
        }
        summary.Total++
        switch f.Severity {
        case "critical":
            summary.Critical++
        case "high":
            summary.High++
        case "medium":
            summary.Medium++
        case "low":
            summary.Low++
        }
    }
    return summary
}