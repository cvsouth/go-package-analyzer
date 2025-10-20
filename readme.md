# Go Package Analyzer

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cvsouth/go-package-analyzer)
[![Tests](https://github.com/cvsouth/go-package-analyzer/actions/workflows/test.yml/badge.svg)](https://github.com/cvsouth/go-package-analyzer/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/cvsouth/go-package-analyzer)](https://goreportcard.com/report/github.com/cvsouth/go-package-analyzer)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/cvsouth/go-package-analyzer/badge)](https://scorecard.dev/viewer/?uri=github.com/cvsouth/go-package-analyzer)

A simple tool to analyze and visualize Go package dependencies.

## Setup

1. **Clone the repository**
   ```bash
   git clone git@github.com:cvsouth/go-package-analyzer.git
   cd go-package-analyzer
   ```

2. **Run the application**
   ```bash
   go run cmd/server.go
   ```

## Usage

Open `http://localhost:6333`.

## Screenshot

![screenshot](https://raw.githubusercontent.com/cvsouth/go-package-analyzer/refs/heads/main/screenshot.png)

## Development

### Static analysis

This project uses [golangci-lint](https://golangci-lint.run/) for code linting. If you have it installed locally, you can run:

```bash
golangci-lint run
```

### Testing

To run the test suite:

```bash
go test ./...
```

## Coming soon

- [ ] Improved readme usage docs
- [ ] Customizable styling
- [ ] Export diagram as SVG / PNG / drawio file
- [ ] Each package in the graph having its public interface displayed in some sort of collapsible list
