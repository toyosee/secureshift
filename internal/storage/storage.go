package storage

import (
    "encoding/json"
    "fmt"
    "secureshift/internal/models"
    "sort"
    "sync"
    "time"

    "go.etcd.io/bbolt"
)

type Storage struct {
    db *bbolt.DB
    mu sync.RWMutex
}

func NewStorage(path string) (*Storage, error) {
    db, err := bbolt.Open(path, 0600, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Create buckets
    err = db.Update(func(tx *bbolt.Tx) error {
        buckets := []string{"scans", "findings", "settings"}
        for _, bucket := range buckets {
            if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
                return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
            }
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
    return s.db.Close()
}

func (s *Storage) SaveScan(result *models.ScanResult) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    return s.db.Update(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("scans"))
        
        data, err := json.Marshal(result)
        if err != nil {
            return err
        }
        
        return bucket.Put([]byte(result.ID), data)
    })
}

func (s *Storage) GetScan(id string) (*models.ScanResult, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var result models.ScanResult
    err := s.db.View(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("scans"))
        data := bucket.Get([]byte(id))
        if data == nil {
            return fmt.Errorf("scan not found")
        }
        return json.Unmarshal(data, &result)
    })
    
    return &result, err
}

func (s *Storage) IgnoreFinding(findingID string, reason string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    return s.db.Update(func(tx *bbolt.Tx) error {
        findingsBucket := tx.Bucket([]byte("findings"))
        finding := map[string]interface{}{
            "ignored": true,
            "reason":  reason,
            "ignored_at": time.Now(),
        }
        data, err := json.Marshal(finding)
        if err != nil {
            return err
        }
        return findingsBucket.Put([]byte(findingID), data)
    })
}

func (s *Storage) GetDashboardStats() (map[string]interface{}, error) {
    stats := map[string]interface{}{
        "total_scans":     0,
        "total_findings":  0,
        "critical":        0,
        "high":            0,
        "medium":          0,
        "low":             0,
    }

    err := s.db.View(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("scans"))
        if bucket == nil {
            return nil
        }

        cursor := bucket.Cursor()
        for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
            var result models.ScanResult
            if err := json.Unmarshal(v, &result); err != nil {
                continue
            }
            
            stats["total_scans"] = stats["total_scans"].(int) + 1
            
            // Count only non-ignored findings
            for _, f := range result.Findings {
                if f.Ignored {
                    continue
                }
                stats["total_findings"] = stats["total_findings"].(int) + 1
                switch f.Severity {
                case "critical":
                    stats["critical"] = stats["critical"].(int) + 1
                case "high":
                    stats["high"] = stats["high"].(int) + 1
                case "medium":
                    stats["medium"] = stats["medium"].(int) + 1
                case "low":
                    stats["low"] = stats["low"].(int) + 1
                }
            }
        }
        return nil
    })

    return stats, err
}

func (s *Storage) GetScanHistory(days int) ([]map[string]interface{}, error) {
    var history []map[string]interface{}
    cutoff := time.Now().AddDate(0, 0, -days)

    err := s.db.View(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("scans"))
        if bucket == nil {
            return nil
        }

        cursor := bucket.Cursor()
        for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
            var result models.ScanResult
            if err := json.Unmarshal(v, &result); err != nil {
                continue
            }

            if result.StartTime.After(cutoff) {
                // Count non-ignored findings
                totalFindings := 0
                criticalFindings := 0
                for _, f := range result.Findings {
                    if !f.Ignored {
                        totalFindings++
                        if f.Severity == "critical" {
                            criticalFindings++
                        }
                    }
                }
                
                history = append(history, map[string]interface{}{
                    "id":       result.ID,
                    "date":     result.StartTime,
                    "findings": totalFindings,
                    "critical": criticalFindings,
                    "status":   result.Status,
                    "project":  result.ProjectName,
                })
            }
        }
        return nil
    })

    // Sort by date descending (most recent first)
    sort.Slice(history, func(i, j int) bool {
        dateI, okI := history[i]["date"].(time.Time)
        dateJ, okJ := history[j]["date"].(time.Time)
        if okI && okJ {
            return dateI.After(dateJ)
        }
        return false
    })

    return history, err
}

// GetRecentFindings returns the most recent findings across all scans
func (s *Storage) GetRecentFindings(limit int) ([]models.Finding, error) {
    var findings []models.Finding

    err := s.db.View(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("scans"))
        if bucket == nil {
            return nil
        }

        cursor := bucket.Cursor()
        // Iterate backwards to get most recent first
        for k, v := cursor.Last(); k != nil && len(findings) < limit; k, v = cursor.Prev() {
            var result models.ScanResult
            if err := json.Unmarshal(v, &result); err != nil {
                continue
            }
            
            // Add critical findings from this scan
            for _, finding := range result.Findings {
                if len(findings) >= limit {
                    break
                }
                if !finding.Ignored {
                    findings = append(findings, finding)
                }
            }
        }
        return nil
    })

    return findings, err
}