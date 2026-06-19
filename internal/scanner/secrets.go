package scanner

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "secureshift/internal/models"
    "strings"
	"log"

    "github.com/google/uuid"
)

type secretPattern struct {
    Pattern  *regexp.Regexp
    Type     string
    Severity string
    Title    string
}

func (s *Scanner) scanSecrets(files []string) []models.Finding {
    var findings []models.Finding
    patterns := getSecretPatterns()

    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            log.Printf("⚠️ Failed to read file %s: %v", file, err)
            continue
        }

        lines := strings.Split(string(content), "\n")
        for i, line := range lines {
            for _, pattern := range patterns {
                if matches := pattern.Pattern.FindAllString(line, -1); len(matches) > 0 {
                    // Skip if it's a comment or test
                    if isCommentLine(line) || isTestFile(file) {
                        continue
                    }

                    findings = append(findings, models.Finding{
                        ID:          uuid.New().String(),
                        File:        file,
                        Line:        i + 1,
                        Severity:    pattern.Severity,
                        Type:        "secret",
                        Title:       pattern.Title,
                        Description: fmt.Sprintf("Found potential %s in code", pattern.Type),
                        Suggestion:  "Remove this secret and use environment variables or a secrets manager",
                        Ignored:     false,
                    })
                }
            }
        }
    }

    return findings
}

func getSecretPatterns() []secretPattern {
    return []secretPattern{
        {
            Pattern:  regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
            Type:     "AWS Access Key",
            Severity: "critical",
            Title:    "AWS Access Key ID found",
        },
        {
            Pattern:  regexp.MustCompile(`(?i)secret["']?\s*[:=]\s*["'][^"']{8,}["']`),
            Type:     "Generic Secret",
            Severity: "high",
            Title:    "Potential secret key found",
        },
        {
            Pattern:  regexp.MustCompile(`(?i)password["']?\s*[:=]\s*["'][^"']{4,}["']`),
            Type:     "Password",
            Severity: "high",
            Title:    "Hardcoded password detected",
        },
        {
            Pattern:  regexp.MustCompile(`(?i)api[_\-]?key["']?\s*[:=]\s*["'][^"']{8,}["']`),
            Type:     "API Key",
            Severity: "critical",
            Title:    "API Key found in code",
        },
        {
            Pattern:  regexp.MustCompile(`github[-_]?token["']?\s*[:=]\s*["'][^"']{8,}["']`),
            Type:     "GitHub Token",
            Severity: "critical",
            Title:    "GitHub token detected",
        },
        {
            Pattern:  regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
            Type:     "OpenAI API Key",
            Severity: "critical",
            Title:    "OpenAI API key found",
        },
        {
            Pattern:  regexp.MustCompile(`-----BEGIN [A-Z]+ PRIVATE KEY-----`),
            Type:     "Private Key",
            Severity: "critical",
            Title:    "Private key found in code",
        },
        {
            Pattern:  regexp.MustCompile(`mongodb://[^/]+:[^@]+@`),
            Type:     "MongoDB Connection String",
            Severity: "critical",
            Title:    "MongoDB connection string with credentials",
        },
        {
            Pattern:  regexp.MustCompile(`postgresql://[^:]+:[^@]+@`),
            Type:     "PostgreSQL Connection String",
            Severity: "critical",
            Title:    "PostgreSQL connection string with credentials",
        },
        {
            Pattern:  regexp.MustCompile(`redis://[^:]+:[^@]+@`),
            Type:     "Redis Connection String",
            Severity: "critical",
            Title:    "Redis connection string with credentials",
        },
    }
}

func isCommentLine(line string) bool {
    trimmed := strings.TrimSpace(line)
    return strings.HasPrefix(trimmed, "//") ||
        strings.HasPrefix(trimmed, "#") ||
        strings.HasPrefix(trimmed, "--") ||
        strings.HasPrefix(trimmed, "/*") ||
        strings.HasPrefix(trimmed, "*")
}

func isTestFile(file string) bool {
    base := filepath.Base(file)
    testPatterns := []string{"_test.go", "test.js", ".test.js", "_test.py", "spec.js", ".spec.js", "_test.rb"}
    for _, pattern := range testPatterns {
        if strings.Contains(base, pattern) {
            return true
        }
    }
    return false
}