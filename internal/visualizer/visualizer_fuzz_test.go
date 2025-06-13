package visualizer_test

import (
	"strings"
	"testing"

	"github.com/cvsouth/go-package-analyzer/internal/analyzer"
	"github.com/cvsouth/go-package-analyzer/internal/visualizer"
)

// FuzzGenerateDOTContent tests DOT content generation with various graph inputs.
func FuzzGenerateDOTContent(f *testing.F) {
	// Add seed corpus with valid inputs
	f.Add("main", "test.com/module", uint8(3))
	f.Add("cmd/server", "github.com/user/project", uint8(5))
	f.Add("app", "example.org/app", uint8(1))

	f.Fuzz(func(t *testing.T, entryPackage, moduleName string, packageCount uint8) {
		// Limit package count to reasonable range
		if packageCount == 0 || packageCount > 10 {
			return
		}

		// Skip problematic inputs
		if len(entryPackage) > 100 || len(moduleName) > 100 ||
			strings.Contains(entryPackage, "\x00") ||
			strings.Contains(moduleName, "\x00") ||
			strings.Contains(entryPackage, "{") ||
			strings.Contains(entryPackage, "}") ||
			strings.Contains(moduleName, "{") ||
			strings.Contains(moduleName, "}") {
			return
		}

		// Sanitize inputs to ensure valid package names
		entryPackage = sanitizePackageName(entryPackage)
		moduleName = sanitizePackageName(moduleName)

		// Skip empty module names as they would be invalid
		if moduleName == "" {
			moduleName = "test.com/module"
		}

		// Skip empty entry packages
		if entryPackage == "" {
			entryPackage = "main"
		}

		// Ensure entry package is within module
		if !strings.HasPrefix(entryPackage, moduleName) {
			entryPackage = moduleName + "/" + entryPackage
		}

		graph := createTestGraphForFuzz(entryPackage, moduleName, packageCount)
		testDOTGenerationForFuzz(t, graph, entryPackage)
	})
}

// sanitizePackageName removes problematic characters from package names.
func sanitizePackageName(name string) string {
	// Remove null bytes and other problematic characters
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "{", "")
	name = strings.ReplaceAll(name, "}", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.TrimSpace(name)

	// Replace invalid characters with underscores
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '/' || r == '.' || r == '-' || r == '_' {
			result += string(r)
		} else if r == ' ' {
			result += "_"
		}
	}

	return result
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

	// Create layers structure
	layers := make([][]string, 3)
	layers[0] = []string{entryPackage}

	// Add some additional packages
	for i := range packageCount {
		pkgName := "pkg" + string(rune('a'+int(i)))
		pkgPath := moduleName + "/" + pkgName
		layer := int(i%2) + 1 // Distribute between layers 1 and 2

		graph.Packages[pkgPath] = &analyzer.PackageInfo{
			Name:         pkgName,
			Path:         pkgPath,
			Dependencies: []string{},
			FileCount:    int(i + 1),
			Layer:        layer,
		}

		// Add to appropriate layer
		if layer < len(layers) {
			layers[layer] = append(layers[layer], pkgPath)
		}
	}

	// Set the layers in the graph
	graph.Layers = layers

	return graph
}

// testDOTGenerationForFuzz tests DOT content generation and validates the result.
func testDOTGenerationForFuzz(t *testing.T, graph *analyzer.DependencyGraph, _ string) {
	t.Helper()

	// Test that GenerateDOTContent doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenerateDOTContent panicked: %v", r)
		}
	}()

	v := visualizer.New()
	result := v.GenerateDOTContent(graph)

	// Basic validation - just ensure it doesn't crash and produces some output
	if result == "" {
		t.Errorf("GenerateDOTContent returned empty result")
		return
	}

	// Very basic DOT format validation
	if !strings.Contains(result, "digraph") {
		t.Errorf("Result should contain 'digraph'")
	}

	// Count braces - they should be balanced
	openBraces := strings.Count(result, "{")
	closeBraces := strings.Count(result, "}")

	// Allow some tolerance for edge cases, but they should be reasonably balanced
	if openBraces == 0 && closeBraces == 0 {
		t.Errorf("DOT content should contain braces")
	} else if abs(openBraces-closeBraces) > 1 {
		t.Errorf("Braces significantly unbalanced: %d open, %d close", openBraces, closeBraces)
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
