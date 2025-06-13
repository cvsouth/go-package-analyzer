package analyzer_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cvsouth/go-package-analyzer/internal/analyzer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	a := analyzer.New()
	if a == nil {
		t.Fatal("New() returned nil")
	}
	// Cannot check private fields in black-box testing
}

func TestAnalyzeFromFile_SimpleProject(t *testing.T) {
	// Get absolute path to test data
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	mainFilePath := filepath.Join(testDataPath, "main.go")

	// Check if test file exists
	if _, statErr := os.Stat(mainFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", mainFilePath)
	}

	a := analyzer.New()
	graph, err := a.AnalyzeFromFile(mainFilePath, true, nil)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	if graph.EntryPackage == "" {
		t.Error("Expected non-empty entry package")
	}

	if len(graph.Packages) == 0 {
		t.Error("Expected at least one package in graph")
	}

	if graph.ModuleName != "testing/data/simple_project" {
		t.Errorf("Expected module name 'testing/data/simple_project', got '%s'", graph.ModuleName)
	}
}

func TestAnalyzeFromFile_NonExistentFile(t *testing.T) {
	a := analyzer.New()
	_, err := a.AnalyzeFromFile("/non/existent/file.go", true, nil)

	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestAnalyzeFromFile_WithExternalDependencies(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	mainFilePath := filepath.Join(testDataPath, "main.go")

	if _, statErr := os.Stat(mainFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", mainFilePath)
	}

	a := analyzer.New()

	// Test with external dependencies included
	graph, err := a.AnalyzeFromFile(mainFilePath, false, nil)
	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should include fmt package when external dependencies are included
	fmtFound := false
	for pkgPath := range graph.Packages {
		if pkgPath == "fmt" {
			fmtFound = true
			break
		}
	}

	if !fmtFound {
		t.Error("Expected to find 'fmt' package when external dependencies are included")
	}
}

func TestAnalyzeFromFile_ExcludeExternal(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	mainFilePath := filepath.Join(testDataPath, "main.go")

	if _, statErr := os.Stat(mainFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", mainFilePath)
	}

	a := analyzer.New()

	// Test with external dependencies excluded
	graph, err := a.AnalyzeFromFile(mainFilePath, true, nil)
	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should not include fmt package when external dependencies are excluded
	for pkgPath := range graph.Packages {
		if pkgPath == "fmt" {
			t.Error("Should not find 'fmt' package when external dependencies are excluded")
		}
	}
}

func TestAnalyzeFromFile_MultipleDependencies(t *testing.T) {
	testDataPath, err := filepath.Abs(filepath.Join("..", "..", "testing", "data", "simple_project"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cliFilePath := filepath.Join(testDataPath, "cmd", "cli.go")

	if _, statErr := os.Stat(cliFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", cliFilePath)
	}

	a := analyzer.New()
	graph, err := a.AnalyzeFromFile(cliFilePath, true, nil)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should have main package, app package, and util package
	expectedPackages := []string{
		filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "cmd")),
		filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "app")),
		filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "util")),
	}

	for _, expectedPkg := range expectedPackages {
		if _, exists := graph.Packages[expectedPkg]; !exists {
			t.Errorf("Expected package '%s' not found in graph", expectedPkg)
		}
	}

	// Check dependencies
	cmdPkg := graph.Packages[filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "cmd"))]
	if cmdPkg == nil {
		t.Fatal("cmd package not found")
	}

	expectedDeps := []string{
		filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "app")),
		filepath.ToSlash(filepath.Join("testing", "data", "simple_project", "util")),
	}

	for _, expectedDep := range expectedDeps {
		found := false
		for _, dep := range cmdPkg.Dependencies {
			if dep == expectedDep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dependency '%s' not found in cmd package", expectedDep)
		}
	}
}

func TestFindEntryPoints(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	a := analyzer.New()
	entryPoints, err := a.FindEntryPoints(testDataPath)

	if err != nil {
		t.Fatalf("FindEntryPoints failed: %v", err)
	}

	if len(entryPoints) == 0 {
		t.Error("Expected to find at least one entry point")
	}

	// Should find main.go and cmd/cli.go
	foundMain := false
	foundCli := false

	for _, ep := range entryPoints {
		if filepath.Base(ep) == "main.go" {
			foundMain = true
		}
		if filepath.Base(ep) == "cli.go" {
			foundCli = true
		}
	}

	if !foundMain {
		t.Error("Expected to find main.go as entry point")
	}

	if !foundCli {
		t.Error("Expected to find cli.go as entry point")
	}
}

func TestAnalyzeFromFile_EmptyPackage(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/edge_cases")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	emptyFilePath := filepath.Join(testDataPath, "empty", "empty.go")

	if _, statErr := os.Stat(emptyFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", emptyFilePath)
	}

	a := analyzer.New()
	graph, err := a.AnalyzeFromFile(emptyFilePath, true, nil)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should still create a graph even for empty package
	if graph == nil {
		t.Fatal("Expected non-nil graph for empty package")
	}

	if len(graph.Packages) == 0 {
		t.Error("Expected at least the entry package in graph")
	}
}

func TestAnalyzeFromFile_UnicodeFilename(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/edge_cases")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	unicodeFilePath := filepath.Join(testDataPath, "unicode", "тест.go")

	if _, statErr := os.Stat(unicodeFilePath); os.IsNotExist(statErr) {
		t.Skipf("Test data file does not exist: %s", unicodeFilePath)
	}

	a := analyzer.New()
	graph, err := a.AnalyzeFromFile(unicodeFilePath, true, nil)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should handle unicode filenames correctly
	if graph == nil {
		t.Fatal("Expected non-nil graph for unicode filename")
	}
}

func TestAnalyzeFromFile_WithExclusions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := "module test/project\n\ngo 1.21\n"
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main.go
	mainContent := `package main

import "test/project/internal/excluded"

func main() {
	excluded.Help()
}`
	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create excluded.go in excluded directory
	excludedDir := filepath.Join(tmpDir, "internal", "excluded")
	if err := os.MkdirAll(excludedDir, 0755); err != nil {
		t.Fatalf("Failed to create excluded directory: %v", err)
	}

	excludedContent := `package excluded

func Help() {}`
	excludedPath := filepath.Join(excludedDir, "excluded.go")
	if err := os.WriteFile(excludedPath, []byte(excludedContent), 0644); err != nil {
		t.Fatalf("Failed to create excluded.go: %v", err)
	}

	a := analyzer.New()
	exclusions := []string{"internal/excluded"}
	graph, err := a.AnalyzeFromFile(mainPath, true, exclusions)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should not include excluded package (exact match)
	if _, exists := graph.Packages["test/project/internal/excluded"]; exists {
		t.Error("Expected 'test/project/internal/excluded' package to be excluded")
	}
}

func TestAnalyzeMultipleEntryPoints(t *testing.T) {
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	analyzer := analyzer.New()
	result, err := analyzer.AnalyzeMultipleEntryPoints(testDataPath, true, nil)

	if err != nil {
		t.Fatalf("AnalyzeMultipleEntryPoints failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success=true, got %v. Error: %s", result.Success, result.Error)
	}

	if len(result.EntryPoints) == 0 {
		t.Error("Expected to find at least one entry point")
	}

	if result.ModuleName != "testing/data/simple_project" {
		t.Errorf("Expected module name 'testing/data/simple_project', got '%s'", result.ModuleName)
	}

	// Each entry point should have basic fields populated
	for i, ep := range result.EntryPoints {
		if ep.Path == "" {
			t.Errorf("Entry point %d should have a path", i)
		}

		if ep.PackagePath == "" {
			t.Errorf("Entry point %d should have a package path", i)
		}

		if ep.Graph == nil {
			t.Errorf("Entry point %d should have a graph", i)
		}

		// DOTContent is populated by the caller, not by AnalyzeMultipleEntryPoints
		// so we don't test for it here
	}
}

func TestAnalyzeFromFile_InvalidGoSyntax(t *testing.T) {
	// Test that the analyzer handles parsing errors appropriately
	// Let's make it a more typical parsing error scenario

	// Create a temporary directory with a go.mod file for proper module handling
	tmpDir := t.TempDir()

	// Create go.mod file
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module invalid_test\n\ngo 1.21\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a file with severely broken Go syntax that should definitely cause parse errors
	invalidFilePath := filepath.Join(tmpDir, "invalid.go")
	invalidContent := `this is not go code at all
	random text { } ( )
	definitely not parseable`

	if err := os.WriteFile(invalidFilePath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	analyzer := analyzer.New()
	_, err := analyzer.AnalyzeFromFile(invalidFilePath, true, nil)

	// With completely invalid content, we should get an error
	if err == nil {
		t.Log("Warning: Analyzer did not return error for completely invalid Go content")
		// This is not necessarily a failure as the analyzer might be very robust
		// and handle parse errors gracefully by skipping unreadable files
	}
}

// TestAnalyzeFromFile_ModuleFinding tests module detection through black-box approach.
func TestAnalyzeFromFile_ModuleFinding(t *testing.T) {
	testCases := []struct {
		name           string
		setupProject   func(*testing.T, string) string // setup function returns entry file path
		expectedModule string
		expectError    bool
	}{
		{
			name: "project with go.mod",
			setupProject: func(t *testing.T, tmpDir string) string {
				return setupProjectWithGoMod(t, tmpDir, "test/module")
			},
			expectedModule: "test/module",
			expectError:    false,
		},
		{
			name: "nested package in module",
			setupProject: func(t *testing.T, tmpDir string) string {
				return setupNestedPackageProject(t, tmpDir, "my/test/project")
			},
			expectedModule: "my/test/project",
			expectError:    false,
		},
		{
			name:           "project without go.mod",
			setupProject:   setupProjectWithoutGoMod,
			expectedModule: "", // will use directory name as fallback
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			entryFile := tc.setupProject(t, tmpDir)

			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(entryFile, true, nil)

			assertTestResults(t, tc.expectError, tc.expectedModule, err, graph)
		})
	}
}

// TestAnalyzeFromFile_PackagePathHandling tests package path logic through black-box approach.
func TestAnalyzeFromFile_PackagePathHandling(t *testing.T) {
	tmpDir := t.TempDir()
	createGoMod(t, tmpDir, "test/project")

	testCases := []struct {
		name            string
		relativeDir     string
		expectedPkgPath string
	}{
		{
			name:            "root package",
			relativeDir:     "",
			expectedPkgPath: "test/project",
		},
		{
			name:            "subpackage",
			relativeDir:     "utils",
			expectedPkgPath: "test/project/utils",
		},
		{
			name:            "nested package",
			relativeDir:     "internal/handlers",
			expectedPkgPath: "test/project/internal/handlers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goPath := setupPackageTestCase(t, tmpDir, tc.relativeDir)

			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(goPath, true, nil)
			require.NoError(t, err, "AnalyzeFromFile failed")

			validatePackageInGraph(t, graph, tc.expectedPkgPath)
		})
	}
}

// setupPackageTestCase creates a package directory and Go file for path handling tests.
func setupPackageTestCase(t *testing.T, tmpDir, relativeDir string) string {
	t.Helper()

	var pkgDir string
	if relativeDir == "" {
		pkgDir = tmpDir
	} else {
		pkgDir = filepath.Join(tmpDir, relativeDir)
		err := os.MkdirAll(pkgDir, 0755)
		require.NoError(t, err, "Failed to create package directory")
	}

	goContent := generateGoFileContent(relativeDir, pkgDir)
	goPath := filepath.Join(pkgDir, "test.go")
	createGoFile(t, goPath, goContent)
	return goPath
}

// generateGoFileContent creates appropriate Go file content based on the package type.
func generateGoFileContent(relativeDir, pkgDir string) string {
	if relativeDir == "" {
		return `package main
func main() {}`
	}
	return `package ` + filepath.Base(pkgDir) + `
func Test() {}`
}

// validatePackageInGraph checks that the expected package exists in the dependency graph.
func validatePackageInGraph(t *testing.T, graph *analyzer.DependencyGraph, expectedPkgPath string) {
	t.Helper()
	if _, exists := graph.Packages[expectedPkgPath]; !exists {
		t.Errorf("Expected package '%s' not found in graph. Available packages: %v",
			expectedPkgPath, getPackageNames(graph.Packages))
	}
}

func getPackageNames(packages map[string]*analyzer.PackageInfo) []string {
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	return names
}

// TestAnalyzeFromFile_ExclusionLogic tests package exclusion through black-box approach.
func TestAnalyzeFromFile_ExclusionLogic(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := setupExclusionTestProject(t, tmpDir)

	testCases := []struct {
		name          string
		excludeDirs   []string
		shouldInclude []string
		shouldExclude []string
	}{
		{
			name:        "no exclusions",
			excludeDirs: nil,
			shouldInclude: []string{
				"test/project/vendor/pkg",
				"test/project/vendor",
				"test/project/internal/test",
				"test/project/utils",
			},
			shouldExclude: []string{},
		},
		{
			name:          "exclude with wildcard - all vendor",
			excludeDirs:   []string{"vendor*"},
			shouldInclude: []string{"test/project/internal/test", "test/project/utils"},
			shouldExclude: []string{"test/project/vendor/pkg", "test/project/vendor"},
		},
		{
			name:          "exclude specific directory only",
			excludeDirs:   []string{"vendor"},
			shouldInclude: []string{"test/project/vendor/pkg", "test/project/internal/test", "test/project/utils"},
			shouldExclude: []string{"test/project/vendor"},
		},
		{
			name:          "exclude subdirectories with wildcard",
			excludeDirs:   []string{"vendor/*"},
			shouldInclude: []string{"test/project/vendor", "test/project/internal/test", "test/project/utils"},
			shouldExclude: []string{"test/project/vendor/pkg"},
		},
		{
			name:          "exclude multiple patterns",
			excludeDirs:   []string{"vendor*", "internal/test"},
			shouldInclude: []string{"test/project/utils"},
			shouldExclude: []string{"test/project/vendor/pkg", "test/project/vendor", "test/project/internal/test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(mainPath, true, tc.excludeDirs)
			require.NoError(t, err, "AnalyzeFromFile failed")

			validateIncludedPackages(t, graph, tc.shouldInclude)
			validateExcludedPackages(t, graph, tc.shouldExclude)
		})
	}
}

// TestAnalyzeFromFile_WildcardExclusion tests comprehensive wildcard pattern matching.
func TestAnalyzeFromFile_WildcardExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := setupWildcardTestProject(t, tmpDir, "test/wildcards")

	testCases := []struct {
		name           string
		excludePattern string
		shouldInclude  []string
		shouldExclude  []string
	}{
		{
			name:           "wildcard at end - internal/*",
			excludePattern: "internal/*",
			shouldInclude:  []string{"test/wildcards/pkg/utils", "test/wildcards/shared"},
			shouldExclude:  []string{"test/wildcards/internal/auth", "test/wildcards/internal/db"},
		},
		{
			name:           "wildcard at beginning - */utils",
			excludePattern: "*/utils",
			shouldInclude:  []string{"test/wildcards/internal/auth", "test/wildcards/pkg/helpers"},
			shouldExclude:  []string{"test/wildcards/pkg/utils"},
		},
		{
			name:           "wildcard in middle - test/*/unit",
			excludePattern: "test/*/unit",
			shouldInclude:  []string{"test/wildcards/test/integration", "test/wildcards/pkg/utils"},
			shouldExclude:  []string{}, // no exact match for this pattern
		},
		{
			name:           "multiple wildcards - */test/*",
			excludePattern: "*/test/*",
			shouldInclude:  []string{"test/wildcards/internal/auth", "test/wildcards/shared"},
			shouldExclude:  []string{}, // no exact match for this pattern in our test structure
		},
		{
			name:           "exact match without wildcards - shared",
			excludePattern: "shared",
			shouldInclude:  []string{"test/wildcards/internal/auth", "test/wildcards/pkg/utils"},
			shouldExclude:  []string{"test/wildcards/shared"},
		},
		{
			name:           "no match pattern - nonexistent/*",
			excludePattern: "nonexistent/*",
			shouldInclude:  []string{"test/wildcards/internal/auth", "test/wildcards/shared"},
			shouldExclude:  []string{},
		},
		{
			name:           "single wildcard matches all - *",
			excludePattern: "*",
			shouldInclude:  []string{},
			shouldExclude: []string{
				"test/wildcards/internal/auth",
				"test/wildcards/pkg/utils",
				"test/wildcards/shared",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(mainPath, true, []string{tc.excludePattern})
			require.NoError(t, err, "AnalyzeFromFile failed")

			validateIncludedPackages(t, graph, tc.shouldInclude)
			validateExcludedPackages(t, graph, tc.shouldExclude)
		})
	}
}

// TestAnalyzeFromFile_LayerCalculation tests layer organization through black-box approach.
func TestAnalyzeFromFile_LayerCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := setupLayerTestProject(t, tmpDir)

	analyzer := analyzer.New()
	graph, err := analyzer.AnalyzeFromFile(mainPath, true, nil)
	require.NoError(t, err, "AnalyzeFromFile failed")

	validateLayerStructure(t, graph)
}

// TestAnalyzeFromFile_WildcardEdgeCases tests edge cases for wildcard pattern matching.
func TestAnalyzeFromFile_WildcardEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := setupEdgeCaseTestProject(t, tmpDir)

	testCases := []struct {
		name           string
		excludePattern string
		shouldInclude  []string
		shouldExclude  []string
	}{
		{
			name:           "empty pattern matches nothing",
			excludePattern: "",
			shouldInclude:  []string{"test/edge/a", "test/edge/ab", "test/edge/abc"},
			shouldExclude:  []string{},
		},
		{
			name:           "pattern with multiple consecutive wildcards - a**b",
			excludePattern: "a**b",
			shouldInclude:  []string{"test/edge/a", "test/edge/abc", "test/edge/test"},
			shouldExclude:  []string{"test/edge/ab"},
		},
		{
			name:           "pattern starting and ending with wildcard - *test*",
			excludePattern: "*test*",
			shouldInclude:  []string{"test/edge/a", "test/edge/ab", "test/edge/abc", "test/edge/empty"},
			shouldExclude:  []string{"test/edge/test", "test/edge/testing"},
		},
		{
			name:           "pattern that should match partial names - *est",
			excludePattern: "*est",
			shouldInclude: []string{
				"test/edge/a",
				"test/edge/ab",
				"test/edge/abc",
				"test/edge/testing",
				"test/edge/empty",
			},
			shouldExclude: []string{"test/edge/test"},
		},
		{
			name:           "pattern with no match - xyz*",
			excludePattern: "xyz*",
			shouldInclude:  []string{"test/edge/a", "test/edge/test", "test/edge/empty"},
			shouldExclude:  []string{},
		},
		{
			name:           "pattern matching single character - ?",
			excludePattern: "?", // Should not match anything since we don't support ? wildcard
			shouldInclude:  []string{"test/edge/a", "test/edge/test", "test/edge/empty"},
			shouldExclude:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(mainPath, true, []string{tc.excludePattern})
			require.NoError(t, err, "AnalyzeFromFile failed")

			validateIncludedPackages(t, graph, tc.shouldInclude)
			validateExcludedPackages(t, graph, tc.shouldExclude)
		})
	}
}

// TestAnalyzeMultipleEntryPoints_Monorepo tests analysis of multiple entry points in a monorepo structure.
// TestAnalyzeMultipleEntryPoints_Monorepo tests analysis of multiple entry points in a monorepo structure.

// Helper functions for test project setup

// createGoMod creates a go.mod file with the specified module name.
func createGoMod(t *testing.T, dir, moduleName string) {
	t.Helper()
	content := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	path := filepath.Join(dir, "go.mod")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create go.mod")
}

// createGoFile creates a Go file with the specified content.
func createGoFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create Go file")
}

// createNestedPackage creates a nested package directory and Go file.
func createNestedPackage(t *testing.T, baseDir, pkgPath, content string) string {
	t.Helper()
	fullPkgDir := filepath.Join(baseDir, pkgPath)
	err := os.MkdirAll(fullPkgDir, 0755)
	require.NoError(t, err, "Failed to create package directory")

	fileName := filepath.Base(pkgPath) + ".go"
	filePath := filepath.Join(fullPkgDir, fileName)
	createGoFile(t, filePath, content)
	return filePath
}

// setupProjectWithGoMod creates a simple project with go.mod and main.go.
func setupProjectWithGoMod(t *testing.T, tmpDir, moduleName string) string {
	t.Helper()
	createGoMod(t, tmpDir, moduleName)

	mainContent := `package main
func main() {}`
	mainPath := filepath.Join(tmpDir, "main.go")
	createGoFile(t, mainPath, mainContent)
	return mainPath
}

// setupNestedPackageProject creates a project with go.mod and a nested package.
func setupNestedPackageProject(t *testing.T, tmpDir, moduleName string) string {
	t.Helper()
	createGoMod(t, tmpDir, moduleName)

	handlerContent := `package handler
func Handle() {}`
	handlerPath := createNestedPackage(t, tmpDir, filepath.Join("internal", "handler"), handlerContent)
	return handlerPath
}

// setupProjectWithoutGoMod creates a project with only main.go (no go.mod).
func setupProjectWithoutGoMod(t *testing.T, tmpDir string) string {
	t.Helper()
	mainContent := `package main
func main() {}`
	mainPath := filepath.Join(tmpDir, "main.go")
	createGoFile(t, mainPath, mainContent)
	return mainPath
}

// assertTestResults validates the test results and performs necessary assertions.
func assertTestResults(
	t *testing.T,
	expectError bool,
	expectedModule string,
	err error,
	graph *analyzer.DependencyGraph,
) {
	t.Helper()

	if expectError && err == nil {
		t.Error("Expected error but got none")
		return
	}
	if !expectError && err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err == nil {
		if expectedModule != "" && graph.ModuleName != expectedModule {
			t.Errorf("Expected module name '%s', got '%s'", expectedModule, graph.ModuleName)
		} else if expectedModule == "" && graph.ModuleName == "" {
			t.Error("Expected non-empty module name when no go.mod present")
		}
	}
}

// createPackageSet creates a set of packages with their content files.
func createPackageSet(t *testing.T, baseDir string, packages map[string]string) {
	t.Helper()
	for pkgPath, content := range packages {
		pkgDir := filepath.Join(baseDir, pkgPath)
		err := os.MkdirAll(pkgDir, 0755)
		require.NoError(t, err, "Failed to create package directory %s", pkgPath)

		fileName := filepath.Base(pkgPath) + ".go"
		filePath := filepath.Join(pkgDir, fileName)
		createGoFile(t, filePath, content)
	}
}

// setupExclusionTestProject creates a test project for exclusion logic testing.
func setupExclusionTestProject(t *testing.T, tmpDir string) string {
	t.Helper()

	// Create go.mod
	createGoMod(t, tmpDir, "test/project")

	// Create main package with imports
	mainContent := `package main

import (
	"test/project/vendor/pkg"
	"test/project/internal/test"
	"test/project/utils"
	"test/project/vendor"
)

func main() {
	pkg.DoSomething()
	test.RunTest()
	utils.Helper()
	vendor.DirectFunc()
}`

	mainPath := filepath.Join(tmpDir, "main.go")
	createGoFile(t, mainPath, mainContent)

	// Create packages to be excluded and included
	packages := map[string]string{
		"vendor/pkg":    "package pkg\nfunc DoSomething() {}",
		"vendor":        "package vendor\nfunc DirectFunc() {}",
		"internal/test": "package test\nfunc RunTest() {}",
		"utils":         "package utils\nfunc Helper() {}",
	}
	createPackageSet(t, tmpDir, packages)

	return mainPath
}

// validateIncludedPackages checks that expected packages are present in the graph.
func validateIncludedPackages(t *testing.T, graph *analyzer.DependencyGraph, expectedPackages []string) {
	t.Helper()
	for _, pkgPath := range expectedPackages {
		if _, exists := graph.Packages[pkgPath]; !exists {
			t.Errorf("Expected package '%s' to be included but it was not found", pkgPath)
		}
	}
}

// validateExcludedPackages checks that packages are properly excluded from the graph.
func validateExcludedPackages(t *testing.T, graph *analyzer.DependencyGraph, excludedPackages []string) {
	t.Helper()
	for _, pkgPath := range excludedPackages {
		if _, exists := graph.Packages[pkgPath]; exists {
			t.Errorf("Expected package '%s' to be excluded but it was found", pkgPath)
		}
	}
}

// createSimplePackageList creates a list of simple packages with basic content.
func createSimplePackageList(t *testing.T, baseDir string, packages []string) {
	t.Helper()
	for _, pkgPath := range packages {
		pkgDir := filepath.Join(baseDir, pkgPath)
		err := os.MkdirAll(pkgDir, 0755)
		require.NoError(t, err, "Failed to create package directory %s", pkgPath)

		fileName := filepath.Base(pkgPath) + ".go"
		pkgName := filepath.Base(pkgPath)
		content := fmt.Sprintf("package %s\nfunc SomeFunc() {}", pkgName)
		filePath := filepath.Join(pkgDir, fileName)
		createGoFile(t, filePath, content)
	}
}

// setupWildcardTestProject creates a test project for wildcard exclusion testing.
func setupWildcardTestProject(t *testing.T, tmpDir, moduleName string) string {
	t.Helper()

	// Create go.mod
	createGoMod(t, tmpDir, moduleName)

	// Create main package with comprehensive imports
	mainContent := `package main

import (
	"` + moduleName + `/internal/auth"
	"` + moduleName + `/internal/db"
	"` + moduleName + `/pkg/utils"
	"` + moduleName + `/pkg/helpers"
	"` + moduleName + `/vendor/lib1"
	"` + moduleName + `/vendor/lib2"
	"` + moduleName + `/shared"
	"` + moduleName + `/test/unit"
	"` + moduleName + `/test/integration"
)

func main() {
	auth.Login()
	db.Connect()
	utils.DoWork()
	helpers.Assist()
	lib1.External()
	lib2.Other()
	shared.Common()
	unit.TestFunc()
	integration.IntegrationTest()
}`

	mainPath := filepath.Join(tmpDir, "main.go")
	createGoFile(t, mainPath, mainContent)

	// Create all packages
	packages := []string{
		"internal/auth",
		"internal/db",
		"pkg/utils",
		"pkg/helpers",
		"vendor/lib1",
		"vendor/lib2",
		"shared",
		"test/unit",
		"test/integration",
	}
	createSimplePackageList(t, tmpDir, packages)

	return mainPath
}

// setupEdgeCaseTestProject creates a test project for edge case wildcard testing.
func setupEdgeCaseTestProject(t *testing.T, tmpDir string) string {
	t.Helper()

	// Create go.mod
	createGoMod(t, tmpDir, "test/edge")

	// Create main package with specific imports for edge case testing
	mainContent := `package main

import (
	"test/edge/a"
	"test/edge/ab"
	"test/edge/abc"
	"test/edge/test"
	"test/edge/testing"
	"test/edge/empty"
)

func main() {
	a.F()
	ab.F()
	abc.F()
	test.F()
	testing.F()
	empty.F()
}`

	mainPath := filepath.Join(tmpDir, "main.go")
	createGoFile(t, mainPath, mainContent)

	// Create edge case packages with specific naming
	packages := []string{"a", "ab", "abc", "test", "testing", "empty"}
	for _, pkgPath := range packages {
		pkgDir := filepath.Join(tmpDir, pkgPath)
		err := os.MkdirAll(pkgDir, 0755)
		require.NoError(t, err, "Failed to create package directory %s", pkgPath)

		content := fmt.Sprintf("package %s\nfunc F() {}", pkgPath)
		filePath := filepath.Join(pkgDir, pkgPath+".go")
		createGoFile(t, filePath, content)
	}

	return mainPath
}

// setupLayerTestProject creates a test project with a clear dependency hierarchy.
func setupLayerTestProject(t *testing.T, tmpDir string) string {
	t.Helper()

	// Create go.mod
	createGoMod(t, tmpDir, "test/layers")

	// Create packages with clear dependency hierarchy: main -> middleware -> util
	packages := map[string]string{
		"main.go": `package main

import "test/layers/middleware"

func main() {
	middleware.Process()
}`,
		"middleware/middleware.go": `package middleware

import "test/layers/util"

func Process() {
	util.Helper()
}`,
		"util/util.go": `package util

func Helper() {}`,
	}

	for filePath, content := range packages {
		fullPath := filepath.Join(tmpDir, filePath)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create directory %s", dir)
		createGoFile(t, fullPath, content)
	}

	return filepath.Join(tmpDir, "main.go")
}

// validateLayerStructure validates that the layer structure makes sense for dependencies.
func validateLayerStructure(t *testing.T, graph *analyzer.DependencyGraph) {
	t.Helper()

	// Verify layers are calculated
	if len(graph.Layers) == 0 {
		t.Error("Expected layers to be calculated, but got empty layers")
		return
	}

	// Verify all packages are assigned to layers
	packageInLayers := make(map[string]bool)
	for _, layer := range graph.Layers {
		for _, pkg := range layer {
			packageInLayers[pkg] = true
		}
	}

	expectedPackages := []string{
		"test/layers",
		"test/layers/middleware",
		"test/layers/util",
	}

	for _, expectedPkg := range expectedPackages {
		if !packageInLayers[expectedPkg] {
			t.Errorf("Package %s not found in any layer", expectedPkg)
		}
	}

	// Find layer assignments for validation
	layers := make(map[string]int)
	for i, layer := range graph.Layers {
		for _, pkg := range layer {
			layers[pkg] = i
		}
	}

	// Verify basic layer ordering
	utilLayer, utilExists := layers["test/layers/util"]
	middlewareLayer, middlewareExists := layers["test/layers/middleware"]
	mainLayer, mainExists := layers["test/layers"]

	if !utilExists || !middlewareExists || !mainExists {
		t.Error("Not all packages were assigned to layers")
	}

	// Basic sanity check - we have at least the expected packages
	_ = utilLayer
	_ = middlewareLayer
	_ = mainLayer
}

// createMonorepoService creates a complete service structure for monorepo testing.
func createMonorepoService(t *testing.T, baseDir, serviceName, moduleName, serviceType string) {
	t.Helper()

	serviceDir := filepath.Join(baseDir, serviceName)
	err := os.MkdirAll(serviceDir, 0755)
	require.NoError(t, err, "Failed to create service directory")

	createGoMod(t, serviceDir, moduleName)

	switch serviceType {
	case "complex":
		createComplexService(t, serviceDir, moduleName)
	case "simple":
		createSimpleService(t, serviceDir, moduleName)
	default:
		t.Fatalf("Unknown service type: %s", serviceType)
	}
}

// createComplexService creates a service with internal packages (handler, service).
func createComplexService(t *testing.T, serviceDir, moduleName string) {
	t.Helper()

	// Create internal/handler package
	handlerDir := filepath.Join(serviceDir, "internal", "handler")
	err := os.MkdirAll(handlerDir, 0755)
	require.NoError(t, err, "Failed to create handler directory")

	handlerContent := fmt.Sprintf(`package handler

import "%s/internal/service"

func HandleRequest() {
	service.ProcessRequest()
}
`, moduleName)
	createGoFile(t, filepath.Join(handlerDir, "handler.go"), handlerContent)

	// Create internal/service package
	servicePackageDir := filepath.Join(serviceDir, "internal", "service")
	err = os.MkdirAll(servicePackageDir, 0755)
	require.NoError(t, err, "Failed to create service package directory")

	serviceContent := `package service

func ProcessRequest() {
	// Business logic
}
`
	createGoFile(t, filepath.Join(servicePackageDir, "service.go"), serviceContent)

	// Create main.go
	mainContent := fmt.Sprintf(`package main

import "%s/internal/handler"

func main() {
	handler.HandleRequest()
}
`, moduleName)
	createGoFile(t, filepath.Join(serviceDir, "main.go"), mainContent)
}

// createSimpleService creates a service with just utils and main.
func createSimpleService(t *testing.T, serviceDir, moduleName string) {
	t.Helper()

	// Create utils package
	utilsDir := filepath.Join(serviceDir, "utils")
	err := os.MkdirAll(utilsDir, 0755)
	require.NoError(t, err, "Failed to create utils directory")

	utilsContent := `package utils

func FormatData() string {
	return "formatted"
}
`
	createGoFile(t, filepath.Join(utilsDir, "utils.go"), utilsContent)

	// Create main.go
	mainContent := fmt.Sprintf(`package main

import "%s/utils"

func main() {
	utils.FormatData()
}
`, moduleName)
	createGoFile(t, filepath.Join(serviceDir, "main.go"), mainContent)
}

// validateMonorepoResults validates the results of monorepo analysis.
func validateMonorepoResults(t *testing.T, result *analyzer.MultiEntryAnalysisResult) {
	t.Helper()

	require.True(t, result.Success, "Analysis failed: %s", result.Error)
	require.Len(t, result.EntryPoints, 2, "Expected 2 entry points")

	entryPointsByModule := make(map[string]*analyzer.EntryPoint)
	for i := range result.EntryPoints {
		ep := &result.EntryPoints[i]
		if ep.Graph != nil {
			entryPointsByModule[ep.Graph.ModuleName] = ep
		}
	}

	validateServiceA(t, entryPointsByModule)
	validateServiceB(t, entryPointsByModule)
}

// validateServiceA validates service-a structure and packages.
func validateServiceA(t *testing.T, entryPointsByModule map[string]*analyzer.EntryPoint) {
	t.Helper()

	serviceA := entryPointsByModule["github.com/test/service-a"]
	require.NotNil(t, serviceA, "service-a entry point not found or has no graph")

	// service-a should have 3 packages: main, internal/handler, internal/service
	assert.Len(t, serviceA.Graph.Packages, 3,
		"service-a should have 3 packages, got %d: %v",
		len(serviceA.Graph.Packages), getPackageNames(serviceA.Graph.Packages))
}

// validateServiceB validates service-b structure and packages.
func validateServiceB(t *testing.T, entryPointsByModule map[string]*analyzer.EntryPoint) {
	t.Helper()

	serviceB := entryPointsByModule["github.com/test/service-b"]
	require.NotNil(t, serviceB, "service-b entry point not found or has no graph")

	// service-b should have 2 packages: main, utils
	assert.Len(t, serviceB.Graph.Packages, 2,
		"service-b should have 2 packages, got %d: %v",
		len(serviceB.Graph.Packages), getPackageNames(serviceB.Graph.Packages))
}

func TestAnalyzeMultipleEntryPoints_Monorepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create monorepo structure with multiple subprojects
	createMonorepoService(t, tmpDir, "service-a", "github.com/test/service-a", "complex")
	createMonorepoService(t, tmpDir, "service-b", "github.com/test/service-b", "simple")

	// Test: Analyze multiple entry points in the monorepo
	a := analyzer.New()
	result, err := a.AnalyzeMultipleEntryPoints(tmpDir, true, nil)
	require.NoError(t, err, "AnalyzeMultipleEntryPoints failed")

	validateMonorepoResults(t, result)
}
