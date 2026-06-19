package models

import "time"

type ProgressUpdate struct {
    ScanID     string `json:"scan_id"`
    Status     string `json:"status"`
    Progress   int    `json:"progress"`
    Message    string `json:"message"`
    TotalFiles int    `json:"total_files,omitempty"`
    Scanned    int    `json:"scanned,omitempty"`
    Findings   int    `json:"findings,omitempty"`
}

type ScanResult struct {
    ID           string      `json:"id"`
    ProjectName  string      `json:"project_name"`
    Status       string      `json:"status"`
    StartTime    time.Time   `json:"start_time"`
    EndTime      time.Time   `json:"end_time"`
    Findings     []Finding   `json:"findings"`
    Summary      Summary     `json:"summary"`
    FilesScanned int         `json:"files_scanned"`
}

type Finding struct {
    ID           string `json:"id"`
    File         string `json:"file"`
    Line         int    `json:"line"`
    Severity     string `json:"severity"` // critical, high, medium, low
    Type         string `json:"type"`     // secret, dependency, code, etc.
    Title        string `json:"title"`
    Description  string `json:"description"`
    Suggestion   string `json:"suggestion"`
    Ignored      bool   `json:"ignored"`
    IgnoreReason string `json:"ignore_reason,omitempty"`
}

type Summary struct {
    Critical int `json:"critical"`
    High     int `json:"high"`
    Medium   int `json:"medium"`
    Low      int `json:"low"`
    Total    int `json:"total"`
}