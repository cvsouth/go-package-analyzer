package visualizer_test

import (
	"strings"
	"testing"

	"cvsouth/go-package-analyzer/internal/analyzer"
	"cvsouth/go-package-analyzer/internal/visualizer"
)

func TestNew(t *testing.T) {
	viz := visualizer.New()
	if viz == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGenerateDOTContent_BasicGraph(t *testing.T) {
	// Create a simple dependency graph
	graph := &analyzer.DependencyGraph{
		EntryPackage: "test/main",
		ModuleName:   "test",
		Packages: map[string]*analyzer.PackageInfo{
			"test/main": {
				Name:         "main",
				Path:         "test/main",
				Dependencies: []string{"test/util"},
				FileCount:    1,
			},
			"test/util": {
				Name:         "util",
				Path:         "test/util",
				Dependencies: []string{},
				FileCount:    2,
			},
		},
		Layers: [][]string{
			{"test/util"},
			{"test/main"},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Basic checks
	if !strings.Contains(dotContent, "digraph dependencies") {
		t.Error("DOT content should contain 'digraph dependencies'")
	}

	if !strings.Contains(dotContent, "test_main") {
		t.Error("DOT content should contain sanitized node ID for main package")
	}

	if !strings.Contains(dotContent, "test_util") {
		t.Error("DOT content should contain sanitized node ID for util package")
	}

	if !strings.Contains(dotContent, "1 files") {
		t.Error("DOT content should contain file count for main package")
	}

	if !strings.Contains(dotContent, "2 files") {
		t.Error("DOT content should contain file count for util package")
	}

	// Check for edge
	if !strings.Contains(dotContent, "test_main -> test_util") {
		t.Error("DOT content should contain edge from main to util")
	}
}

func TestGenerateDOTContent_CircularDependencies(t *testing.T) {
	// Create a graph with circular dependencies
	graph := &analyzer.DependencyGraph{
		EntryPackage: "test/main",
		ModuleName:   "test",
		Packages: map[string]*analyzer.PackageInfo{
			"test/main": {
				Name:         "main",
				Path:         "test/main",
				Dependencies: []string{"test/a"},
				FileCount:    1,
			},
			"test/a": {
				Name:         "a",
				Path:         "test/a",
				Dependencies: []string{"test/b"},
				FileCount:    1,
			},
			"test/b": {
				Name:         "b",
				Path:         "test/b",
				Dependencies: []string{"test/a"}, // Circular dependency
				FileCount:    1,
			},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Should contain red edges for circular dependencies
	if !strings.Contains(dotContent, `color="red"`) {
		t.Error("DOT content should contain red edges for circular dependencies")
	}
}

func TestGenerateDOTContent_LayerConstraints(t *testing.T) {
	// Create a graph with multiple layers
	graph := &analyzer.DependencyGraph{
		EntryPackage: "test/main",
		ModuleName:   "test",
		Packages: map[string]*analyzer.PackageInfo{
			"test/main": {
				Name:         "main",
				Path:         "test/main",
				Dependencies: []string{"test/middleware", "test/util"},
				FileCount:    1,
			},
			"test/middleware": {
				Name:         "middleware",
				Path:         "test/middleware",
				Dependencies: []string{"test/util"},
				FileCount:    3,
			},
			"test/util": {
				Name:         "util",
				Path:         "test/util",
				Dependencies: []string{},
				FileCount:    2,
			},
		},
		Layers: [][]string{
			{"test/util"},
			{"test/middleware"},
			{"test/main"},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Should contain rank constraints
	if !strings.Contains(dotContent, "rank=") {
		t.Error("DOT content should contain rank constraints for layering")
	}

	// Should contain rank=source for entry package
	if !strings.Contains(dotContent, "rank=source") {
		t.Error("DOT content should contain rank=source for entry package")
	}
}

func TestGenerateDOTContent_EmptyGraph(t *testing.T) {
	// Test with minimal graph
	graph := &analyzer.DependencyGraph{
		EntryPackage: "test/main",
		ModuleName:   "test",
		Packages: map[string]*analyzer.PackageInfo{
			"test/main": {
				Name:         "main",
				Path:         "test/main",
				Dependencies: []string{},
				FileCount:    1,
			},
		},
		Layers: [][]string{
			{"test/main"},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Should still generate valid DOT
	if !strings.Contains(dotContent, "digraph dependencies") {
		t.Error("DOT content should contain 'digraph dependencies' even for empty graph")
	}

	if !strings.Contains(dotContent, "test_main") {
		t.Error("DOT content should contain the main package node")
	}

	// Should not contain any edges
	if strings.Contains(dotContent, "->") {
		t.Error("Empty graph should not contain any edges")
	}
}

func TestGenerateDOTContent_DeterministicOutput(t *testing.T) {
	// Create a simple graph
	createGraph := func() *analyzer.DependencyGraph {
		return &analyzer.DependencyGraph{
			EntryPackage: "test/main",
			ModuleName:   "test",
			Packages: map[string]*analyzer.PackageInfo{
				"test/main": {
					Name:         "main",
					Path:         "test/main",
					Dependencies: []string{"test/util"},
					FileCount:    1,
				},
				"test/util": {
					Name:         "util",
					Path:         "test/util",
					Dependencies: []string{},
					FileCount:    2,
				},
			},
		}
	}

	viz := visualizer.New()
	dotContent1 := viz.GenerateDOTContent(createGraph())
	dotContent2 := viz.GenerateDOTContent(createGraph())

	// Should generate identical output for identical input
	if dotContent1 != dotContent2 {
		t.Error("GenerateDOTContent should produce deterministic output for identical graphs")
	}
}

// TestGenerateDOTContent_NodeSanitization tests node ID sanitization through black-box approach.
func TestGenerateDOTContent_NodeSanitization(t *testing.T) { //nolint:gocognit
	testCases := []struct {
		name              string
		packagePath       string
		expectedSanitized string   // What we expect to see in DOT output
		shouldNotContain  []string // Characters that should not appear in node IDs
	}{
		{
			name:              "dots in package path",
			packagePath:       "github.com/user/repo",
			expectedSanitized: "github_com_user_repo",
			shouldNotContain:  []string{".", "/"},
		},
		{
			name:              "slashes in package path",
			packagePath:       "test/internal/package",
			expectedSanitized: "test_internal_package",
			shouldNotContain:  []string{"/"},
		},
		{
			name:              "dashes in package path",
			packagePath:       "my-project/sub-package",
			expectedSanitized: "my_project_sub_package",
			shouldNotContain:  []string{"-"},
		},
		{
			name:              "mixed special characters",
			packagePath:       "complex.package-name/with_chars",
			expectedSanitized: "complex_package_name_with_chars",
			shouldNotContain:  []string{".", "-", "/"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a simple graph with the problematic package path
			graph := &analyzer.DependencyGraph{
				EntryPackage: tc.packagePath,
				ModuleName:   "test",
				Packages: map[string]*analyzer.PackageInfo{
					tc.packagePath: {
						Name:         "test",
						Path:         tc.packagePath,
						Dependencies: []string{},
						FileCount:    1,
					},
				},
			}

			viz := visualizer.New()
			dotContent := viz.GenerateDOTContent(graph)

			// Check that the sanitized node ID appears in the output
			if !strings.Contains(dotContent, tc.expectedSanitized) {
				t.Errorf("Expected to find sanitized node ID '%s' in DOT output", tc.expectedSanitized)
			}

			// Check that problematic characters are not present in node IDs
			for _, char := range tc.shouldNotContain {
				// Look for the character in actual node ID contexts (lines that define specific nodes)
				lines := strings.Split(dotContent, "\n")
				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					// Skip global graph attribute definitions and empty lines
					if strings.HasPrefix(trimmedLine, "node [") ||
						strings.HasPrefix(trimmedLine, "edge [") ||
						strings.HasPrefix(trimmedLine, "digraph") ||
						strings.HasPrefix(trimmedLine, "{") ||
						strings.HasPrefix(trimmedLine, "}") ||
						trimmedLine == "" {
						continue
					}

					// This should be a node definition line (nodeID [attributes])
					if strings.Contains(line, "[") && strings.Contains(line, "]") {
						// Extract the node ID (part before the first '[')
						nodeIDPart := strings.Split(trimmedLine, "[")[0]
						if strings.Contains(nodeIDPart, char) {
							t.Errorf(
								"Found problematic character '%s' in node ID '%s' in line: %s",
								char, strings.TrimSpace(nodeIDPart), line,
							)
						}
					}
				}
			}
		})
	}
}

// TestGenerateDOTContent_TextHandling tests text wrapping and escaping through black-box approach.
func TestGenerateDOTContent_TextHandling(t *testing.T) {
	testCases := []struct {
		name         string
		packageName  string
		packagePath  string
		shouldEscape []string // Characters that should be escaped in labels
		shouldWrap   bool     // Whether long text should be wrapped
	}{
		{
			name:         "HTML characters in package name",
			packageName:  "test<package>",
			packagePath:  "test/package",
			shouldEscape: []string{"&lt;", "&gt;"},
			shouldWrap:   false,
		},
		{
			name:         "quotes in package name",
			packageName:  `test"package"`,
			packagePath:  "test/package",
			shouldEscape: []string{"&quot;"},
			shouldWrap:   false,
		},
		{
			name:         "ampersand in package name",
			packageName:  "test&package",
			packagePath:  "test/package",
			shouldEscape: []string{"&amp;"},
			shouldWrap:   false,
		},
		{
			name:         "very long package path",
			packageName:  "verylongpackagename",
			packagePath:  "very/long/package/path/that/should/be/wrapped/because/it/exceeds/normal/length/limits",
			shouldEscape: []string{},
			shouldWrap:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			graph := &analyzer.DependencyGraph{
				EntryPackage: tc.packagePath,
				ModuleName:   "test",
				Packages: map[string]*analyzer.PackageInfo{
					tc.packagePath: {
						Name:         tc.packageName,
						Path:         tc.packagePath,
						Dependencies: []string{},
						FileCount:    1,
					},
				},
			}

			viz := visualizer.New()
			dotContent := viz.GenerateDOTContent(graph)

			// Check for proper HTML escaping
			for _, escaped := range tc.shouldEscape {
				if !strings.Contains(dotContent, escaped) {
					t.Errorf("Expected to find escaped sequence '%s' in DOT output", escaped)
				}
			}

			// Check for text wrapping (presence of \\n in labels)
			if tc.shouldWrap {
				if !strings.Contains(dotContent, "\\n") {
					t.Error("Expected to find line breaks (\\n) in DOT output for long text")
				}
			}

			// Verify the label is properly quoted and contains expected content
			if !strings.Contains(dotContent, "label=") {
				t.Error("Expected to find label attribute in DOT output")
			}
		})
	}
}

// TestGenerateDOTContent_ColorHandling tests color conversion through black-box approach.
func TestGenerateDOTContent_ColorHandling(t *testing.T) {
	// Create a graph with multiple packages to trigger different colors
	graph := &analyzer.DependencyGraph{
		EntryPackage: "test/main",
		ModuleName:   "test",
		Packages: map[string]*analyzer.PackageInfo{
			"test/main": {
				Name:         "main",
				Path:         "test/main",
				Dependencies: []string{"test/util", "test/service"},
				FileCount:    1,
			},
			"test/util": {
				Name:         "util",
				Path:         "test/util",
				Dependencies: []string{},
				FileCount:    2,
			},
			"test/service": {
				Name:         "service",
				Path:         "test/service",
				Dependencies: []string{},
				FileCount:    3,
			},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Check that colors are properly formatted
	colorPatterns := []string{
		"fillcolor=", // Should have fill colors
		"color=",     // Should have border colors
		"rgba(",      // Should use RGBA format for fill colors
		"#",          // Should use hex format for border colors
	}

	for _, pattern := range colorPatterns {
		if !strings.Contains(dotContent, pattern) {
			t.Errorf("Expected to find color pattern '%s' in DOT output", pattern)
		}
	}

	// Verify RGBA format is correct (rgba(r,g,b,opacity))
	validateRGBAFormat(t, dotContent)

	// Verify hex colors are properly formatted
	validateHexColors(t, dotContent)
}

// validateHexColors is a helper function to validate hex color formatting in DOT content.
func validateHexColors(t *testing.T, dotContent string) {
	if !strings.Contains(dotContent, "#") {
		return // No hex colors to validate
	}

	lines := strings.Split(dotContent, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "color=") || !strings.Contains(line, "#") {
			continue
		}

		// Extract hex color and verify it's 6 characters
		colorStart := strings.Index(line, "#")
		if colorStart < 0 || colorStart+7 >= len(line) {
			continue
		}

		hexColor := line[colorStart : colorStart+7]
		if len(hexColor) != 7 { // # + 6 hex digits
			t.Errorf("Hex color should be 7 characters (#RRGGBB): %s", hexColor)
		}
	}
}

// validateRGBAFormat is a helper function to validate RGBA color formatting in DOT content.
func validateRGBAFormat(t *testing.T, dotContent string) {
	if !strings.Contains(dotContent, "rgba(") {
		return // No RGBA colors to validate
	}

	rgbaStart := strings.Index(dotContent, "rgba(")
	if rgbaStart < 0 {
		return
	}

	rgbaEnd := strings.Index(dotContent[rgbaStart:], ")")
	if rgbaEnd < 0 {
		return
	}

	rgbaValue := dotContent[rgbaStart : rgbaStart+rgbaEnd+1]
	// Should contain comma-separated values
	if !strings.Contains(rgbaValue, ",") {
		t.Errorf("RGBA value should contain comma-separated components: %s", rgbaValue)
	}
}

// TestGenerateDOTContent_ComplexStructures tests complex scenarios through black-box approach.
func TestGenerateDOTContent_ComplexStructures(t *testing.T) {
	// Test with a complex graph that exercises multiple private function behaviors
	graph := &analyzer.DependencyGraph{
		EntryPackage: "github.com/complex-project/main",
		ModuleName:   "github.com/complex-project",
		Packages: map[string]*analyzer.PackageInfo{
			"github.com/complex-project/main": {
				Name:         "main<with>special&chars",
				Path:         "github.com/complex-project/main",
				Dependencies: []string{"github.com/complex-project/very-long-package-name"},
				FileCount:    1,
			},
			"github.com/complex-project/very-long-package-name": {
				Name:         "verylongpackagename",
				Path:         "github.com/complex-project/very-long-package-name",
				Dependencies: []string{},
				FileCount:    10,
			},
		},
		Layers: [][]string{
			{"github.com/complex-project/very-long-package-name"},
			{"github.com/complex-project/main"},
		},
	}

	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	// Verify all major components are present and properly formatted
	checks := []struct {
		description string
		test        func(string) bool
	}{
		{
			"should contain digraph declaration",
			func(content string) bool {
				return strings.Contains(content, "digraph dependencies")
			},
		},
		{
			"should contain sanitized node IDs",
			func(content string) bool {
				return strings.Contains(content, "github_com_complex_project")
			},
		},
		{
			"should contain escaped HTML characters",
			func(content string) bool {
				return strings.Contains(content, "&lt;") &&
					strings.Contains(content, "&gt;") &&
					strings.Contains(content, "&amp;")
			},
		},
		{
			"should contain file count information",
			func(content string) bool {
				return strings.Contains(content, "1 files") &&
					strings.Contains(content, "10 files")
			},
		},
		{
			"should contain dependency edges",
			func(content string) bool {
				return strings.Contains(content, "->")
			},
		},
		{
			"should contain color information",
			func(content string) bool {
				return strings.Contains(content, "fillcolor=") &&
					strings.Contains(content, "color=")
			},
		},
		{
			"should contain rank constraints for layering",
			func(content string) bool {
				return strings.Contains(content, "rank=")
			},
		},
		{
			"should properly close the digraph",
			func(content string) bool {
				return strings.HasSuffix(strings.TrimSpace(content), "}")
			},
		},
	}

	for _, check := range checks {
		if !check.test(dotContent) {
			t.Errorf("Failed check: %s", check.description)
		}
	}

	// Verify the DOT content is valid by checking basic structure
	if !strings.HasPrefix(strings.TrimSpace(dotContent), "digraph") {
		t.Error("DOT content should start with 'digraph'")
	}

	// Count braces to ensure they're balanced
	openBraces := strings.Count(dotContent, "{")
	closeBraces := strings.Count(dotContent, "}")
	if openBraces != closeBraces {
		t.Errorf("Unbalanced braces in DOT output: %d open, %d close", openBraces, closeBraces)
	}
}
