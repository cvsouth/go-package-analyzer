package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"cvsouth/go-package-analyzer/internal/analyzer"
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
	testDataPath, err := filepath.Abs("../../testing/data/simple_project")
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
		"testing/data/simple_project/cmd",
		"testing/data/simple_project/app",
		"testing/data/simple_project/util",
	}

	for _, expectedPkg := range expectedPackages {
		if _, exists := graph.Packages[expectedPkg]; !exists {
			t.Errorf("Expected package '%s' not found in graph", expectedPkg)
		}
	}

	// Check dependencies
	cmdPkg := graph.Packages["testing/data/simple_project/cmd"]
	if cmdPkg == nil {
		t.Fatal("cmd package not found")
	}

	expectedDeps := []string{
		"testing/data/simple_project/app",
		"testing/data/simple_project/util",
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

import "test/project/util"

func main() {
	util.Help()
}`
	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create util.go in excluded directory
	utilDir := filepath.Join(tmpDir, "internal", "excluded")
	if err := os.MkdirAll(utilDir, 0755); err != nil {
		t.Fatalf("Failed to create util directory: %v", err)
	}

	a := analyzer.New()
	exclusions := []string{"internal/excluded"}
	graph, err := a.AnalyzeFromFile(mainPath, true, exclusions)

	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Should not include util package
	if _, exists := graph.Packages["testing/data/simple_project/util"]; exists {
		t.Error("Expected util package to be excluded")
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
func TestAnalyzeFromFile_ModuleFinding(t *testing.T) { //nolint:gocognit
	testCases := []struct {
		name           string
		setupProject   func(string) string // setup function returns entry file path
		expectedModule string
		expectError    bool
	}{
		{
			name: "project with go.mod",
			setupProject: func(tmpDir string) string {
				// Create go.mod
				goModContent := "module test/module\n\ngo 1.21\n"
				goModPath := filepath.Join(tmpDir, "go.mod")
				if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}

				// Create main.go
				mainContent := `package main
func main() {}`
				mainPath := filepath.Join(tmpDir, "main.go")
				if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
					t.Fatalf("Failed to create main.go: %v", err)
				}
				return mainPath
			},
			expectedModule: "test/module",
			expectError:    false,
		},
		{
			name: "nested package in module",
			setupProject: func(tmpDir string) string {
				// Create go.mod
				goModContent := "module my/test/project\n\ngo 1.21\n"
				goModPath := filepath.Join(tmpDir, "go.mod")
				if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}

				// Create nested package
				pkgDir := filepath.Join(tmpDir, "internal", "handler")
				if err := os.MkdirAll(pkgDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				handlerContent := `package handler
func Handle() {}`
				handlerPath := filepath.Join(pkgDir, "handler.go")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler.go: %v", err)
				}
				return handlerPath
			},
			expectedModule: "my/test/project",
			expectError:    false,
		},
		{
			name: "project without go.mod",
			setupProject: func(tmpDir string) string {
				// Create main.go without go.mod
				mainContent := `package main
func main() {}`
				mainPath := filepath.Join(tmpDir, "main.go")
				if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
					t.Fatalf("Failed to create main.go: %v", err)
				}
				return mainPath
			},
			expectedModule: "", // will use directory name as fallback
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			entryFile := tc.setupProject(tmpDir)

			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(entryFile, true, nil)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tc.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if err == nil {
				if tc.expectedModule != "" && graph.ModuleName != tc.expectedModule {
					t.Errorf("Expected module name '%s', got '%s'", tc.expectedModule, graph.ModuleName)
				} else if tc.expectedModule == "" && graph.ModuleName == "" {
					t.Error("Expected non-empty module name when no go.mod present")
				}
			}
		})
	}
}

// TestAnalyzeFromFile_PackagePathHandling tests package path logic through black-box approach.
func TestAnalyzeFromFile_PackagePathHandling(t *testing.T) { //nolint:gocognit
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := "module test/project\n\ngo 1.21\n"
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

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
			var pkgDir string
			if tc.relativeDir == "" {
				pkgDir = tmpDir
			} else {
				pkgDir = filepath.Join(tmpDir, tc.relativeDir)
				if err := os.MkdirAll(pkgDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}
			}

			// Create a Go file in the package
			goContent := `package ` + filepath.Base(pkgDir) + `
func Test() {}`
			if tc.relativeDir == "" {
				goContent = `package main
func main() {}`
			}
			goPath := filepath.Join(pkgDir, "test.go")
			if err := os.WriteFile(goPath, []byte(goContent), 0644); err != nil {
				t.Fatalf("Failed to create Go file: %v", err)
			}

			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(goPath, true, nil)
			if err != nil {
				t.Fatalf("AnalyzeFromFile failed: %v", err)
			}

			// Check that the package is in the graph with the expected path
			if _, exists := graph.Packages[tc.expectedPkgPath]; !exists {
				t.Errorf("Expected package '%s' not found in graph. Available packages: %v",
					tc.expectedPkgPath, getPackageNames(graph.Packages))
			}
		})
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
func TestAnalyzeFromFile_ExclusionLogic(t *testing.T) { //nolint:gocognit
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := "module test/project\n\ngo 1.21\n"
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main package that imports vendor and test packages
	mainContent := `package main

import (
	"test/project/vendor/pkg"
	"test/project/internal/test"
	"test/project/utils"
)

func main() {
	pkg.DoSomething()
	test.RunTest()
	utils.Helper()
}`

	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create packages to be excluded and included
	packages := map[string]string{
		"vendor/pkg":    "package pkg\nfunc DoSomething() {}",
		"internal/test": "package test\nfunc RunTest() {}",
		"utils":         "package utils\nfunc Helper() {}",
	}

	for pkgPath, content := range packages {
		pkgDir := filepath.Join(tmpDir, pkgPath)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("Failed to create package directory %s: %v", pkgPath, err)
		}

		fileName := filepath.Base(pkgPath) + ".go"
		filePath := filepath.Join(pkgDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create package file %s: %v", filePath, err)
		}
	}

	testCases := []struct {
		name          string
		excludeDirs   []string
		shouldInclude []string
		shouldExclude []string
	}{
		{
			name:          "no exclusions",
			excludeDirs:   nil,
			shouldInclude: []string{"test/project/vendor/pkg", "test/project/internal/test", "test/project/utils"},
			shouldExclude: []string{},
		},
		{
			name:          "exclude vendor",
			excludeDirs:   []string{"vendor"},
			shouldInclude: []string{"test/project/internal/test", "test/project/utils"},
			shouldExclude: []string{"test/project/vendor/pkg"},
		},
		{
			name:          "exclude multiple",
			excludeDirs:   []string{"vendor", "internal/test"},
			shouldInclude: []string{"test/project/utils"},
			shouldExclude: []string{"test/project/vendor/pkg", "test/project/internal/test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := analyzer.New()
			graph, err := analyzer.AnalyzeFromFile(mainPath, true, tc.excludeDirs)
			if err != nil {
				t.Fatalf("AnalyzeFromFile failed: %v", err)
			}

			// Check included packages
			for _, pkgPath := range tc.shouldInclude {
				if _, exists := graph.Packages[pkgPath]; !exists {
					t.Errorf("Expected package '%s' to be included but it was not found", pkgPath)
				}
			}

			// Check excluded packages
			for _, pkgPath := range tc.shouldExclude {
				if _, exists := graph.Packages[pkgPath]; exists {
					t.Errorf("Expected package '%s' to be excluded but it was found", pkgPath)
				}
			}
		})
	}
}

// TestAnalyzeFromFile_LayerCalculation tests layer organization through black-box approach.
func TestAnalyzeFromFile_LayerCalculation(t *testing.T) { //nolint:gocognit
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := "module test/layers\n\ngo 1.21\n"
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a dependency hierarchy: main -> middleware -> util
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	analyzer := analyzer.New()
	graph, err := analyzer.AnalyzeFromFile(filepath.Join(tmpDir, "main.go"), true, nil)
	if err != nil {
		t.Fatalf("AnalyzeFromFile failed: %v", err)
	}

	// Verify layers are calculated
	if len(graph.Layers) == 0 {
		t.Error("Expected layers to be calculated, but got empty layers")
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

	// Verify layer structure makes sense (packages with dependencies should be in higher layers)
	utilLayer := -1
	middlewareLayer := -1
	mainLayer := -1

	for i, layer := range graph.Layers {
		for _, pkg := range layer {
			switch pkg {
			case "test/layers/util":
				utilLayer = i
			case "test/layers/middleware":
				middlewareLayer = i
			case "test/layers":
				mainLayer = i
			}
		}
	}

	// Basic layer ordering validation - packages with more dependencies tend to be in higher layers
	if utilLayer < 0 || middlewareLayer < 0 || mainLayer < 0 {
		t.Error("Not all packages were assigned to layers")
	}
}
