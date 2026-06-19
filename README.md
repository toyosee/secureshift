# 🔒 SecureShift

**SecureShift** is a modern, self-contained security scanner that helps developers identify vulnerabilities before shipping code. No installation required - just download and run!

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/toyosee/secureshift)](https://github.com/toyosee/secureshift/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/toyosee/secureshift)](https://goreportcard.com/report/github.com/toyosee/secureshift)

## ✨ Features

- 🔍 **Secrets Detection** - Finds hardcoded API keys, passwords, tokens, and private keys
- 📦 **Dependency Scanning** - Checks for known vulnerabilities in your dependencies (Go, NPM, Python, Maven, Composer, Ruby)
- 🛡️ **Code Analysis** - Identifies common security antipatterns (SQL injection, command injection, path traversal, weak cryptography)
- 📄 **PDF Reports** - Generate professional security reports with detailed findings
- 🎨 **Beautiful Web UI** - Modern interface with dark/light mode support
- 🚀 **Zero Installation** - Single binary, just download and run
- 🔒 **Privacy First** - All scanning happens locally on your machine
- 📊 **Interactive Dashboard** - Visualize scan history and findings trends
- 📁 **Multiple Input Methods** - Upload single files, entire folders, or clone Git repositories

<!-- ## 📸 Screenshots

### Dashboard
![Dashboard](https://via.placeholder.com/800x400/1a2333/4f8cff?text=SecureShift+Dashboard)

### Scan Results
![Scan Results](https://via.placeholder.com/800x400/1a2333/2ed573?text=Scan+Results)

### PDF Report
![PDF Report](https://via.placeholder.com/800x400/1a2333/ff6b35?text=PDF+Report) -->

## 🚀 Quick Start

### Download
Download the latest release for your platform from [Releases](https://github.com/toyosee/secureshift/releases/tag/v1.0.0)

### Run
```bash
# Linux/macOS
chmod +x secure-shift
./secure-shift

# Windows
secure-shift.exe

Open Browser

The web interface will automatically open at: http://localhost:8080
Start Scanning

    Click "New Scan" tab

    Either:

        Drag & drop your project folder

        Upload a ZIP file

        Enter a Git repository URL

    Wait for the scan to complete

    View detailed findings and download PDF reports

📋 How It Works

SecureShift scans your code for:
1. Secrets Detection

    AWS Access Keys (AKIA...)

    OpenAI API Keys (sk-...)

    GitHub Tokens

    Private Keys (RSA, SSH, etc.)

    Database Connection Strings

    Hardcoded Passwords

    Generic API Keys

2. Dependency Scanning

    Checks against the OSV.dev vulnerability database

    Supports:

        Go (go.mod)

        NPM (package.json)

        Python (requirements.txt)

        Maven (pom.xml)

        Composer (composer.json)

        Ruby (Gemfile)

        Cargo (Cargo.toml)

3. Code Analysis

    SQL Injection patterns

    Command Injection vulnerabilities

    Path Traversal

    Weak Cryptographic Algorithms

    Server-Side Request Forgery (SSRF)

    XML External Entity (XXE) vulnerabilities

    Unsafe pointer usage

🖥️ System Requirements

    Operating Systems: Windows 10/11, Linux, macOS

    Memory: ~50MB RAM

    Disk Space: ~20MB for binary

    Git: Required only for Git repository scanning

📦 Download Options
Platform	Download
Windows 64-bit	Download
Windows 32-bit	Download
Linux 64-bit	Download
Linux 32-bit	Download
macOS Intel	Download
macOS Apple Silicon	Download
🛠️ Development
Prerequisites

    Go 1.21 or higher

    Git (for cloning and testing)

Clone Repository
bash

git clone https://github.com/toyosee/secureshift.git
cd secureshift

Install Dependencies
bash

go mod download
go mod tidy

Build from Source
bash

# Build for current platform
go build -o secure-shift main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o secure-shift.exe main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o secure-shift main.go

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o secure-shift main.go

Run in Development
bash

go run main.go

🏗️ Architecture
text

┌─────────────────────────────────────────────┐
│          Single Binary (secure-shift)        │
├─────────────────────────────────────────────┤
│  ┌─────────┐  ┌──────────┐  ┌────────────┐ │
│  │  Web UI  │  │  Scanner │  │  Storage   │ │
│  │ (HTML/CSS/│  │ (Secrets, │  │  (BoltDB)  │ │
│  │   JS)    │  │  Deps,   │  │            │ │
│  └─────────┘  │  Code)   │  └────────────┘ │
│               └──────────┘                  │
│  ┌──────────────────────────────────────┐   │
│  │      Embedded HTTP Server            │   │
│  │      (Go net/http)                  │   │
│  └──────────────────────────────────────┘   │
├─────────────────────────────────────────────┤
│           Single Binary (no deps)           │
└─────────────────────────────────────────────┘

🔒 Security

    No Data Collection - Your code never leaves your machine

    Local Processing - All scanning is performed locally

    No External Dependencies - Everything is included in the binary

    Open Source - Code is transparent and auditable

📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
Ways to Contribute

    🐛 Report bugs and issues

    💡 Suggest new features

    📝 Improve documentation

    🔧 Submit pull requests

    ⭐ Star the repository

Development Workflow

    Fork the repository

    Create a feature branch (git checkout -b feature/amazing-feature)

    Commit your changes (git commit -m 'Add amazing feature')

    Push to the branch (git push origin feature/amazing-feature)

    Open a Pull Request

📊 Roadmap

    AST-based code analysis for more languages

    Automatic fix suggestions with PR creation

    CI/CD integrations (GitHub Actions, GitLab CI)

    Custom rule definitions

    Multi-project dashboard

    API key rotation suggestions

    Container image scanning

🙏 Acknowledgments

    OSV.dev - Vulnerability database

    BoltDB - Embedded database

    Chart.js - Charts and visualization

    Font Awesome - Icons

👨‍💻 Author

    Elijah Abolaji

    GitHub: @toyosee

    LinkedIn: Elijah Abolaji

    Email: tyabolaji@gmail.com

⭐ Support

    If you find SecureShift useful, please give it a ⭐ on GitHub!

# https://img.shields.io/github/stars/toyosee/secureshift?style=social

Happy Coding! 🚀