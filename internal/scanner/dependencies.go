package scanner

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "secureshift/internal/models"
    "strings"
    "time"
	"log"

    "github.com/google/uuid"
)

type Dependency struct {
    Name    string `json:"name"`
    Version string `json:"version"`
    Eco     string `json:"ecosystem"`
}

// OSV types
type OSVResponse struct {
    Vulns []OSVVuln `json:"vulns"`
}

type OSVVuln struct {
    ID       string `json:"id"`
    Summary  string `json:"summary"`
    Severity string `json:"severity"`
    Affected []struct {
        Ranges []struct {
            Events []struct {
                Introduced string `json:"introduced"`
                Fixed      string `json:"fixed"`
            } `json:"events"`
        } `json:"ranges"`
    } `json:"affected"`
}

// Simplified vuln for returning
type SimpleVuln struct {
    ID       string `json:"id"`
    Summary  string `json:"summary"`
    Severity string `json:"severity"`
}

func (s *Scanner) scanDependencies(path string) []models.Finding {
    var findings []models.Finding
    dependencies := s.parseDependencies(path)

    if len(dependencies) == 0 {
        return findings
    }

    log.Printf("📦 Found %d dependencies to check", len(dependencies))

    for i, dep := range dependencies {
        // Update progress for each batch of dependencies
        if i%5 == 0 {
            progress := 65 + (i*15 / len(dependencies))
            s.updateProgress("", "scanning", progress, fmt.Sprintf("Checking dependency %d/%d...", i+1, len(dependencies)))
        }

        vulns := s.checkOSV(dep)
        for _, vuln := range vulns {
            findings = append(findings, models.Finding{
                ID:          uuid.New().String(),
                File:        "dependencies",
                Line:        0,
                Severity:    mapSeverity(vuln.Severity),
                Type:        "dependency",
                Title:       fmt.Sprintf("Vulnerability in %s", dep.Name),
                Description: fmt.Sprintf("%s: %s", vuln.ID, vuln.Summary),
                Suggestion:  fmt.Sprintf("Update %s to a version that fixes this vulnerability", dep.Name),
                Ignored:     false,
            })
        }
    }

    return findings
}

func (s *Scanner) parseDependencies(path string) []Dependency {
    var deps []Dependency

    // Parse go.mod
    goModPath := filepath.Join(path, "go.mod")
    if data, err := os.ReadFile(goModPath); err == nil {
        deps = append(deps, parseGoMod(string(data))...)
    }

    // Parse package.json
    packageJSONPath := filepath.Join(path, "package.json")
    if data, err := os.ReadFile(packageJSONPath); err == nil {
        deps = append(deps, parsePackageJSON(string(data))...)
    }

    // Parse requirements.txt
    reqPath := filepath.Join(path, "requirements.txt")
    if data, err := os.ReadFile(reqPath); err == nil {
        deps = append(deps, parseRequirements(string(data))...)
    }

    return deps
}

func parseGoMod(content string) []Dependency {
    var deps []Dependency
    lines := strings.Split(content, "\n")
    inRequire := false

    for _, line := range lines {
        line = strings.TrimSpace(line)

        if strings.HasPrefix(line, "require (") {
            inRequire = true
            continue
        }
        if inRequire && line == ")" {
            inRequire = false
            continue
        }

        if inRequire && line != "" && !strings.HasPrefix(line, "//") {
            parts := strings.Fields(line)
            if len(parts) >= 2 {
                deps = append(deps, Dependency{
                    Name:    strings.Trim(parts[0], "\""),
                    Version: strings.Trim(parts[1], "\""),
                    Eco:     "go",
                })
            }
        } else if strings.HasPrefix(line, "require") && !strings.Contains(line, "(") {
            parts := strings.Fields(line)
            if len(parts) >= 3 {
                deps = append(deps, Dependency{
                    Name:    strings.Trim(parts[1], "\""),
                    Version: strings.Trim(parts[2], "\""),
                    Eco:     "go",
                })
            }
        }
    }
    return deps
}

func parsePackageJSON(content string) []Dependency {
    var deps []Dependency
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(content), &data); err != nil {
        return deps
    }

    // Check dependencies and devDependencies
    depSections := []string{"dependencies", "devDependencies"}
    for _, section := range depSections {
        if depsObj, ok := data[section].(map[string]interface{}); ok {
            for name, version := range depsObj {
                if versionStr, ok := version.(string); ok {
                    // Clean up version (remove ^, ~, etc.)
                    cleanedVersion := strings.TrimPrefix(versionStr, "^")
                    cleanedVersion = strings.TrimPrefix(cleanedVersion, "~")
                    cleanedVersion = strings.TrimPrefix(cleanedVersion, ">=")
                    cleanedVersion = strings.TrimPrefix(cleanedVersion, "<=")
                    cleanedVersion = strings.Split(cleanedVersion, " ")[0]

                    deps = append(deps, Dependency{
                        Name:    name,
                        Version: cleanedVersion,
                        Eco:     "npm",
                    })
                }
            }
        }
    }

    return deps
}

func parseRequirements(content string) []Dependency {
    var deps []Dependency
    lines := strings.Split(content, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        var name, version string
        if strings.Contains(line, "==") {
            parts := strings.Split(line, "==")
            name = strings.TrimSpace(parts[0])
            version = strings.TrimSpace(parts[1])
        } else if strings.Contains(line, ">=") {
            parts := strings.Split(line, ">=")
            name = strings.TrimSpace(parts[0])
            version = strings.TrimSpace(parts[1])
        } else if strings.Contains(line, "<=") {
            parts := strings.Split(line, "<=")
            name = strings.TrimSpace(parts[0])
            version = strings.TrimSpace(parts[1])
        } else if strings.Contains(line, "~=") {
            parts := strings.Split(line, "~=")
            name = strings.TrimSpace(parts[0])
            version = strings.TrimSpace(parts[1])
        }

        if name != "" && version != "" {
            deps = append(deps, Dependency{
                Name:    name,
                Version: version,
                Eco:     "pypi",
            })
        }
    }
    return deps
}

func (s *Scanner) checkOSV(dep Dependency) []SimpleVuln {
    client := &http.Client{Timeout: 10 * time.Second}

    // Query OSV.dev API
    url := "https://api.osv.dev/v1/query"
    payload := map[string]interface{}{
        "package": map[string]string{
            "name":      dep.Name,
            "ecosystem": dep.Eco,
        },
        "version": dep.Version,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil
    }

    resp, err := client.Post(url, "application/json", strings.NewReader(string(jsonData)))
    if err != nil {
        return nil
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil
    }

    var osvResp OSVResponse
    if err := json.Unmarshal(body, &osvResp); err != nil {
        return nil
    }

    // Convert OSVVuln to SimpleVuln
    var simpleVulns []SimpleVuln
    for _, vuln := range osvResp.Vulns {
        simpleVulns = append(simpleVulns, SimpleVuln{
            ID:       vuln.ID,
            Summary:  vuln.Summary,
            Severity: vuln.Severity,
        })
    }

    return simpleVulns
}

func mapSeverity(severity string) string {
    switch strings.ToLower(severity) {
    case "critical":
        return "critical"
    case "high":
        return "high"
    case "medium":
        return "medium"
    default:
        return "low"
    }
}