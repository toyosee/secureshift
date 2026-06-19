package scanner

import (
    // "fmt"
    "os"
    "regexp"
    "secureshift/internal/models"
    "strings"

    "github.com/google/uuid"
)

type codePattern struct {
    Pattern    *regexp.Regexp
    Type       string
    Severity   string
    Title      string
    Suggestion string
}

func (s *Scanner) scanCodePatterns(files []string) []models.Finding {
    var findings []models.Finding
    patterns := getCodePatterns()

    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            continue
        }

        lines := strings.Split(string(content), "\n")
        for i, line := range lines {
            for _, pattern := range patterns {
                if pattern.Pattern.MatchString(line) {
                    // Skip if it's a comment
                    if isCommentLine(line) {
                        continue
                    }

                    findings = append(findings, models.Finding{
                        ID:          uuid.New().String(),
                        File:        file,
                        Line:        i + 1,
                        Severity:    pattern.Severity,
                        Type:        "code",
                        Title:       pattern.Title,
                        Description: "Security antipattern detected in code",
                        Suggestion:  pattern.Suggestion,
                        Ignored:     false,
                    })
                }
            }
        }
    }

    return findings
}

func getCodePatterns() []codePattern {
    return []codePattern{
        {
            Pattern:    regexp.MustCompile(`(?i)exec\.Command.*\b(bash|sh|cmd|powershell)\b`),
            Type:       "Command Injection",
            Severity:   "critical",
            Title:      "Command injection vulnerability",
            Suggestion: "Use exec.Command with arguments as separate parameters, not a single string",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)sql\.(Query|Exec|Raw|DB\..*Query).*\+`),
            Type:       "SQL Injection",
            Severity:   "critical",
            Title:      "SQL injection vulnerability detected",
            Suggestion: "Use parameterized queries or an ORM with safe query building",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)mysql_query|pg_query|mysqli_query.*\$`),
            Type:       "SQL Injection",
            Severity:   "critical",
            Title:      "SQL injection vulnerability detected",
            Suggestion: "Use prepared statements with bound parameters",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)eval\s*\(.*\$`),
            Type:       "Code Injection",
            Severity:   "critical",
            Title:      "Dangerous eval() usage with user input",
            Suggestion: "Avoid using eval with user-controlled data",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)ioutil\.ReadFile\(.*\+.*\)`),
            Type:       "Path Traversal",
            Severity:   "high",
            Title:      "Potential path traversal vulnerability",
            Suggestion: "Validate and sanitize file paths, use path.Join and check for '..'",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)os\.Open\(.*\+.*\)`),
            Type:       "Path Traversal",
            Severity:   "high",
            Title:      "Potential path traversal vulnerability",
            Suggestion: "Validate and sanitize file paths before opening",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)http\.Get\(.*\$`),
            Type:       "SSRF",
            Severity:   "high",
            Title:      "Potential Server-Side Request Forgery (SSRF)",
            Suggestion: "Validate and whitelist URLs, limit protocols and ports",
        },
        {
            Pattern:    regexp.MustCompile(`(?i)md5\(|sha1\(`),
            Type:       "Weak Hashing",
            Severity:   "medium",
            Title:      "Weak cryptographic hash function used",
            Suggestion: "Use SHA-256 or stronger hashing algorithms for security",
        },
    }
}