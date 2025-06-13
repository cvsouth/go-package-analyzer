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

Open your web browser and navigate to `http://localhost:6333`.

Enter the local path to your Go project in the input field and click "Analyze". The tool will generate a visual representation of the package dependencies.

If your project contains multiple applications such as in the case with a monorepo, enter the root path of the project and you will be able to switch between applications using the dropdown menu on the graph.

## Release Verification

This project uses [SLSA](https://slsa.dev/) (Supply-chain Levels for Software Artifacts) provenance generation to ensure the integrity and authenticity of releases. All release artifacts are cryptographically signed and include verifiable build provenance.

### Verifying Release Integrity

To verify a downloaded release, you can use the [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) tool:

1. **Install slsa-verifier**:
   ```bash
   go install github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@latest
   ```

2. **Download the binary and its provenance** from the [releases page](https://github.com/cvsouth/go-package-analyzer/releases).

3. **Verify the binary**:
   ```bash
   slsa-verifier verify-artifact go-package-analyzer-linux-amd64 \
     --provenance-path go-package-analyzer-linux-amd64.intoto.jsonl \
     --source-uri github.com/cvsouth/go-package-analyzer
   ```

### What This Guarantees

✅ **Authenticity**: The binary was built from the exact source code in this repository  
✅ **Integrity**: The binary hasn't been tampered with since it was built  
✅ **Transparency**: Complete build environment and process information is available  
✅ **Non-repudiation**: Cryptographic proof of the build's origin and time  

The provenance files (`.intoto.jsonl`) contain machine-readable attestations that include:
- Source repository and commit SHA
- Build environment details
- Build command and parameters
- Timestamps and runner information

For more information about SLSA and supply chain security, visit [slsa.dev](https://slsa.dev/).

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
