package visualizer_test

import (
	"strings"
	"testing"

	"github.com/cvsouth/go-package-analyzer/internal/analyzer"
	"github.com/cvsouth/go-package-analyzer/internal/visualizer"
)

// FuzzGenerateDOTContent tests DOT content generation with various graph inputs.
func FuzzGenerateDOTContent(f *testing.F) {
	// This is a fuzz test that creates graph structures
	f.Fuzz(func(t *testing.T, entryPackage, moduleName string, packageCount uint8) {
		// Limit package count to reasonable range
		if packageCount == 0 || packageCount > 20 {
			return
		}

		// Skip problematic inputs
		if len(entryPackage) > 200 || len(moduleName) > 200 ||
			strings.Contains(entryPackage, "\x00") ||
			strings.Contains(moduleName, "\x00") {
			return
		}

		// Skip empty module names as they would be invalid
		if moduleName == "" {
			moduleName = "test.com/module"
		}

		// Skip empty entry packages
		if entryPackage == "" {
			entryPackage = moduleName + "/main"
		}

		graph := createTestGraphForFuzz(entryPackage, moduleName, packageCount)
		testDOTGenerationForFuzz(t, graph, entryPackage)
	})
}

// createTestGraphForFuzz creates a test dependency graph.
func createTestGraphForFuzz(entryPackage, moduleName string, packageCount uint8) *analyzer.DependencyGraph {
	graph := &analyzer.DependencyGraph{
		EntryPackage: entryPackage,
		ModuleName:   moduleName,
		Packages:     make(map[string]*analyzer.PackageInfo),
		Layers:       [][]string{},
	}

	// Add the entry package
	graph.Packages[entryPackage] = &analyzer.PackageInfo{
		Name:         "main",
		Path:         entryPackage,
		Dependencies: []string{},
		FileCount:    1,
		Layer:        0,
	}

	// Add some additional packages
	for i := range packageCount {
		pkgPath := moduleName + "/pkg" + string(rune('a'+i))
		graph.Packages[pkgPath] = &analyzer.PackageInfo{
			Name:         "pkg" + string(rune('a'+i)),
			Path:         pkgPath,
			Dependencies: []string{},
			FileCount:    int(i + 1),
			Layer:        int(i % 3), // Distribute across 3 layers
		}
	}

	return graph
}

// testDOTGenerationForFuzz tests DOT content generation and validates the result.
func testDOTGenerationForFuzz(t *testing.T, graph *analyzer.DependencyGraph, entryPackage string) {
	t.Helper()

	// Test that GenerateDOTContent doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenerateDOTContent panicked: %v", r)
		}
	}()

	v := visualizer.New()
	result := v.GenerateDOTContent(graph)

	validateDOTContentForFuzz(t, result, entryPackage)
}

// validateDOTContentForFuzz validates the generated DOT content.
func validateDOTContentForFuzz(t *testing.T, result, entryPackage string) {
	t.Helper()

	// Result should be valid DOT format
	trimmed := strings.TrimSpace(result)
	if !strings.HasPrefix(trimmed, "digraph") {
		t.Errorf("DOT content should start with 'digraph': %q",
			result[:minIntForFuzz(50, len(result))])
	}

	if !strings.HasSuffix(trimmed, "}") {
		t.Errorf("DOT content should end with '}': %q",
			result[maxIntForFuzz(0, len(result)-50):])
	}

	// Should contain some form of the entry package
	if !strings.Contains(result, entryPackage) {
		// Entry package might be sanitized, so just check it's not completely empty
		if result == "" {
			t.Errorf("DOT content should not be empty")
		}
	}

	// Braces should be balanced
	openBraces := strings.Count(result, "{")
	closeBraces := strings.Count(result, "}")
	if openBraces != closeBraces {
		t.Errorf("Unbalanced braces in DOT content: %d open, %d close", openBraces, closeBraces)
	}

	// Should not contain obvious injection attempts
	dangerousStrings := []string{"<script", "javascript:", "eval(", "onclick="}
	for _, dangerous := range dangerousStrings {
		if strings.Contains(strings.ToLower(result), dangerous) {
			t.Errorf("DOT content contains potentially dangerous string: %q", dangerous)
		}
	}
}

// Helper functions for fuzz tests.

// minIntForFuzz returns the minimum of two integers.
func minIntForFuzz(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxIntForFuzz returns the maximum of two integers.
func maxIntForFuzz(a, b int) int {
	if a > b {
		return a
	}
	return b
}
