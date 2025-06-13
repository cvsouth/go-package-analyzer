package analyzer_test

import (
	"cvsouth/go-package-analyzer/internal/analyzer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzAnalyzeFromFile tests the AnalyzeFromFile function with various file paths.
func FuzzAnalyzeFromFile(f *testing.F) {
	// Add seed corpus with various file path formats
	f.Add("./cmd/server.go", true)
	f.Add("internal/analyzer/analyzer.go", false)
	f.Add("test.go", true)
	f.Add("", false)

	a := analyzer.New()
	f.Fuzz(func(t *testing.T, filePath string, excludeExternal bool) {
		// Test that AnalyzeFromFile doesn't panic with any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AnalyzeFromFile panicked with filePath=%q: %v", filePath, r)
			}
		}()

		// Create a temporary file if the path doesn't exist
		if filePath != "" && !strings.Contains(filePath, "..") {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")
			goModFile := filepath.Join(tmpDir, "go.mod")

			// Create a basic go.mod
			goModContent := "module test\n\ngo 1.21\n"
			if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
				return // Skip if we can't create test files
			}

			// Create a basic Go file
			goContent := "package main\n\nfunc main() {}\n"
			if err := os.WriteFile(testFile, []byte(goContent), 0644); err != nil {
				return
			}

			filePath = testFile
		}

		_, err := a.AnalyzeFromFile(filePath, excludeExternal, nil)

		// Function should handle errors gracefully
		if err != nil {
			// Errors are expected for invalid file paths
			return
		}
	})
}

// FuzzFindEntryPoints tests the FindEntryPoints function with various directory paths.
func FuzzFindEntryPoints(f *testing.F) {
	// Add seed corpus with various directory paths
	f.Add(".")
	f.Add("./cmd")
	f.Add("./internal")
	f.Add("")
	f.Add("/tmp")

	a := analyzer.New()
	f.Fuzz(func(t *testing.T, repoRoot string) {
		// Test that FindEntryPoints doesn't panic with any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("FindEntryPoints panicked with repoRoot=%q: %v", repoRoot, r)
			}
		}()

		// Skip obviously problematic paths
		if strings.Contains(repoRoot, "..") ||
			strings.Contains(repoRoot, "\x00") ||
			len(repoRoot) > 1000 {
			return
		}

		entryPoints, err := a.FindEntryPoints(repoRoot)

		// Function should handle errors gracefully
		if err != nil {
			// Errors are expected for invalid paths
			return
		}

		// Validate returned entry points
		for _, entryPoint := range entryPoints {
			if entryPoint == "" {
				t.Errorf("FindEntryPoints returned empty entry point")
			}

			if !strings.HasSuffix(entryPoint, ".go") {
				t.Errorf("Entry point should be a .go file: %q", entryPoint)
			}
		}
	})
}

// FuzzAnalyzeMultipleEntryPoints tests multi-entry analysis with various parameters.
func FuzzAnalyzeMultipleEntryPoints(f *testing.F) {
	// Add seed corpus
	f.Add(".", true, "vendor,node_modules")
	f.Add("./cmd", false, "")
	f.Add("", true, "test")

	a := analyzer.New()
	f.Fuzz(func(t *testing.T, repoRoot string, excludeExternal bool, excludeDirsStr string) {
		// Test that AnalyzeMultipleEntryPoints doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AnalyzeMultipleEntryPoints panicked: %v", r)
			}
		}()

		// Skip problematic inputs
		if shouldSkipInput(repoRoot, excludeDirsStr) {
			return
		}

		// Parse exclude directories and run analysis
		excludeDirs := parseExcludeDirs(excludeDirsStr)
		result, err := a.AnalyzeMultipleEntryPoints(repoRoot, excludeExternal, excludeDirs)

		// Function should handle errors gracefully
		if err != nil {
			return
		}

		validateAnalysisResult(t, result)
	})
}

// shouldSkipInput checks if inputs should be skipped for fuzzing.
func shouldSkipInput(repoRoot, excludeDirsStr string) bool {
	return strings.Contains(repoRoot, "..") ||
		strings.Contains(repoRoot, "\x00") ||
		len(repoRoot) > 500 ||
		len(excludeDirsStr) > 1000
}

// parseExcludeDirs parses the exclude directories string.
func parseExcludeDirs(excludeDirsStr string) []string {
	var excludeDirs []string
	if excludeDirsStr != "" {
		excludeDirs = strings.Split(excludeDirsStr, ",")
		for i, dir := range excludeDirs {
			excludeDirs[i] = strings.TrimSpace(dir)
		}
	}
	return excludeDirs
}

// validateAnalysisResult validates the analysis result.
func validateAnalysisResult(t *testing.T, result *analyzer.MultiEntryAnalysisResult) {
	// Validate result structure
	if result == nil {
		t.Errorf("AnalyzeMultipleEntryPoints returned nil result")
		return
	}

	// Check result consistency
	if result.Success && result.Error != "" {
		t.Errorf("Result shows success but has error message: %q", result.Error)
	}

	if !result.Success && result.Error == "" {
		t.Errorf("Result shows failure but no error message")
	}
}
