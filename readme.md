# Go Package Analyzer

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cvsouth/go-package-analyzer)
[![Tests](https://github.com/cvsouth/go-package-analyzer/actions/workflows/test.yml/badge.svg)](https://github.com/cvsouth/go-package-analyzer/actions/workflows/test.yml)

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

Open your web browser and navigate to `http://localhost:6333`.

Enter the local path to your Go project in the input field and click "Analyze". The tool will generate a visual representation of the package dependencies.

If your project contains multiple applications such as in the case with a monorepo, enter the root path of the project and you will be able to switch between applications using the dropdown menu on the graph.

## Demo

Clicking the image below will navigate you to YouTube.

[![Go Package Analyzer Demo](https://raw.githubusercontent.com/cvsouth/go-package-analyzer/refs/heads/main/screenshot.png)](https://www.youtube.com/watch?v=_1yVsU9JKJA)

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
