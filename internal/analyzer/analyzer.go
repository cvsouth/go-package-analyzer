// Package analyzer provides functionality for analyzing Go package dependencies.
package analyzer

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Constants for layer calculation.
const (
	maxIterationsPadding = 5 // Additional iterations to ensure layer convergence
)

// Analyzer analyzes Go package dependencies.
type Analyzer struct {
	fileSet     *token.FileSet
	moduleRoot  string
	moduleName  string
	excludeDirs []string
}

// PackageInfo represents information about a Go package.
type PackageInfo struct {
	Name         string
	Path         string
	Dependencies []string
	Layer        int // Layer in the dependency graph (0 = bottom layer)
	FileCount    int // Number of Go files in the package
}

// DependencyGraph represents the package dependency graph.
type DependencyGraph struct {
	EntryPackage string
	Packages     map[string]*PackageInfo
	Layers       [][]string // Packages organized by layer
	ModuleName   string     // Name of the Go module
}

// EntryPoint represents a detected entry point in the codebase.
type EntryPoint struct {
	Path         string           `json:"path"`         // Absolute file path
	RelativePath string           `json:"relativePath"` // Relative path from repository root
	PackagePath  string           `json:"packagePath"`  // Go package path
	DOTContent   string           `json:"dotContent"`   // Generated DOT visualization
	Graph        *DependencyGraph `json:"-"`            // Internal graph data (not serialized)
}

// MultiEntryAnalysisResult represents the result of analyzing multiple entry points.
type MultiEntryAnalysisResult struct {
	Success     bool         `json:"success"`
	EntryPoints []EntryPoint `json:"entryPoints,omitempty"`
	Error       string       `json:"error,omitempty"`
	RepoRoot    string       `json:"repoRoot"`
	ModuleName  string       `json:"moduleName"`
}

// New creates a new analyzer.
func New() *Analyzer {
	return &Analyzer{
		fileSet: token.NewFileSet(),
	}
}

// AnalyzeFromFile analyzes package dependencies starting from a Go file.
func (a *Analyzer) AnalyzeFromFile(
	entryFile string,
	excludeExternal bool,
	excludeDirs []string,
) (*DependencyGraph, error) {
	a.excludeDirs = excludeDirs

	// Always find the correct module for this specific entry file
	// This ensures each entry point in a monorepo uses its correct module context
	if err := a.findModule(entryFile); err != nil {
		// If no go.mod found, use the directory containing the entry file as module root
		entryDir := filepath.Dir(entryFile)
		absEntryDir, absErr := filepath.Abs(entryDir)
		if absErr != nil {
			return nil, fmt.Errorf("resolving entry directory: %w", absErr)
		}
		a.moduleRoot = absEntryDir
		a.moduleName = filepath.Base(absEntryDir)
	}

	// Parse the entry file to get its package
	entryPkg, err := a.getPackageFromFile(entryFile)
	if err != nil {
		return nil, fmt.Errorf("getting entry package: %w", err)
	}

	// Build dependency graph
	graph := &DependencyGraph{
		EntryPackage: entryPkg,
		Packages:     make(map[string]*PackageInfo),
		ModuleName:   a.moduleName,
	}

	// Recursively analyze all packages
	visited := make(map[string]bool)
	if analyzeErr := a.analyzePackage(entryPkg, graph, visited, excludeExternal); analyzeErr != nil {
		return nil, fmt.Errorf("analyzing packages: %w", analyzeErr)
	}

	// Calculate layers
	a.calculateLayers(graph)

	return graph, nil
}

// findModule finds the module root by looking for go.mod file.
func (a *Analyzer) findModule(startPath string) error {
	// Check if startPath is a file or directory
	stat, err := os.Stat(startPath)
	if err != nil {
		return fmt.Errorf("accessing start path: %w", err)
	}

	var dir string
	if stat.IsDir() {
		dir = startPath
	} else {
		dir = filepath.Dir(startPath)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, statErr := os.Stat(goModPath); statErr == nil {
			a.moduleRoot = dir

			// Read module name from go.mod
			content, readErr := os.ReadFile(goModPath)
			if readErr != nil {
				return fmt.Errorf("reading go.mod: %w", readErr)
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					a.moduleName = strings.TrimSpace(line[7:])
					return nil
				}
			}
			return errors.New("module name not found in go.mod")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return errors.New("go.mod not found")
		}
		dir = parent
	}
}

// getPackageFromFile determines the package path from a Go file.
func (a *Analyzer) getPackageFromFile(filePath string) (string, error) {
	// Get relative path from module root
	relPath, err := filepath.Rel(a.moduleRoot, filepath.Dir(filePath))
	if err != nil {
		return "", err
	}

	if relPath == "." {
		return a.moduleName, nil
	}

	return filepath.Join(a.moduleName, relPath), nil
}

// analyzePackage recursively analyzes a package and its dependencies.
func (a *Analyzer) analyzePackage(
	pkgPath string,
	graph *DependencyGraph,
	visited map[string]bool,
	excludeExternal bool,
) error {
	if visited[pkgPath] {
		return nil
	}
	visited[pkgPath] = true

	// Skip excluded directories
	if a.isExcludedPackage(pkgPath) {
		return nil
	}

	// Skip external packages if excludeExternal is true
	if excludeExternal && !a.isInternalPackage(pkgPath) {
		return nil
	}

	// Handle external packages when excludeExternal is false
	if !a.isInternalPackage(pkgPath) {
		// Add external package to graph as a leaf node (no dependencies to analyze)
		pkgInfo := &PackageInfo{
			Name:         a.getPackageName(pkgPath),
			Path:         pkgPath,
			Dependencies: []string{}, // External packages have no analyzable dependencies
			FileCount:    0,          // We can't count files for external packages
		}
		graph.Packages[pkgPath] = pkgInfo
		return nil
	}

	// Get package directory for internal packages
	pkgDir, err := a.getPackageDir(pkgPath)
	if err != nil {
		return fmt.Errorf("getting package directory for %s: %w", pkgPath, err)
	}

	// Parse all Go files in the package
	dependencies, fileCount, err := a.parsePackageImports(pkgDir)
	if err != nil {
		return fmt.Errorf("parsing imports for %s: %w", pkgPath, err)
	}

	// Filter dependencies if needed
	if excludeExternal {
		filtered := make([]string, 0)
		for _, dep := range dependencies {
			if a.isInternalPackage(dep) {
				filtered = append(filtered, dep)
			}
		}
		sort.Strings(filtered) // Sort filtered dependencies for consistency
		dependencies = filtered
	}

	// Create package info
	pkgInfo := &PackageInfo{
		Name:         a.getPackageName(pkgPath),
		Path:         pkgPath,
		Dependencies: dependencies,
		FileCount:    fileCount,
		Layer:        0,
	}
	graph.Packages[pkgPath] = pkgInfo

	// Recursively analyze dependencies
	for _, dep := range dependencies {
		if depErr := a.analyzePackage(dep, graph, visited, excludeExternal); depErr != nil {
			// Log error but continue with other dependencies
			slog.Warn("Warning: failed to analyze dependency",
				"dependency", dep,
				"error", depErr)
		}
	}

	return nil
}

// isInternalPackage checks if a package is internal to the module.
func (a *Analyzer) isInternalPackage(pkgPath string) bool {
	return strings.HasPrefix(pkgPath, a.moduleName)
}

// isExcludedPackage checks if a package should be excluded based on the exclude list.
func (a *Analyzer) isExcludedPackage(pkgPath string) bool {
	if !a.isInternalPackage(pkgPath) {
		return false // Only check exclusions for internal packages
	}

	// Get the relative path from the module root
	relPath := strings.TrimPrefix(pkgPath, a.moduleName)
	relPath = strings.TrimPrefix(relPath, "/")

	// Check if the relative path matches any excluded pattern
	for _, excludePattern := range a.excludeDirs {
		if a.matchesWildcardPattern(relPath, excludePattern) {
			return true
		}
	}

	return false
}

// matchesWildcardPattern checks if a path matches a wildcard pattern.
// The pattern can contain * wildcards which match any sequence of characters.
// If no wildcards are present, it performs exact matching.
func (a *Analyzer) matchesWildcardPattern(path, pattern string) bool {
	// Empty pattern matches nothing
	if pattern == "" {
		return false
	}

	// If pattern contains no wildcards, do exact match
	if !strings.Contains(pattern, "*") {
		return path == pattern
	}

	// Handle wildcard patterns
	return a.wildcardMatch(path, pattern)
}

// wildcardMatch implements wildcard pattern matching where * matches any sequence of characters.
func (a *Analyzer) wildcardMatch(text, pattern string) bool {
	// Convert pattern to regexp-like matching logic
	// Split pattern by * to get literal parts
	parts := strings.Split(pattern, "*")

	// Handle edge case: pattern is just "*"
	if len(parts) == 2 && parts[0] == "" && parts[1] == "" {
		return true // "*" matches everything
	}

	textIndex := 0

	for i, part := range parts {
		if part == "" {
			// This is a wildcard segment (between two *), skip it
			continue
		}

		// Find this part in the remaining text
		index := strings.Index(text[textIndex:], part)
		if index == -1 {
			return false // Part not found
		}

		// Adjust the actual index in the full text
		index += textIndex

		// Special handling for first and last parts
		if i == 0 {
			// First part must match at the beginning (unless preceded by *)
			if index != 0 && pattern[0] != '*' {
				return false
			}
		}

		if i == len(parts)-1 {
			// Last part must match at the end (unless followed by *)
			expectedEnd := index + len(part)
			if expectedEnd != len(text) && !strings.HasSuffix(pattern, "*") {
				return false
			}
		}

		// Move text index past this match
		textIndex = index + len(part)
	}

	return true
}

// getPackageDir converts a package path to a directory path.
func (a *Analyzer) getPackageDir(pkgPath string) (string, error) {
	if !a.isInternalPackage(pkgPath) {
		return "", fmt.Errorf("external package: %s", pkgPath)
	}

	// Remove module name prefix
	relPath := strings.TrimPrefix(pkgPath, a.moduleName)
	relPath = strings.TrimPrefix(relPath, "/")

	if relPath == "" {
		return a.moduleRoot, nil
	}

	return filepath.Join(a.moduleRoot, relPath), nil
}

// parsePackageImports parses all Go files in a directory to extract imports and count files.
func (a *Analyzer) parsePackageImports(dir string) ([]string, int, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}

	importSet := make(map[string]bool)
	fileCount := 0

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}

		fileCount++
		filePath := filepath.Join(dir, file.Name())
		imports, parseErr := a.parseFileImports(filePath)
		if parseErr != nil {
			continue // Skip files that can't be parsed
		}

		for _, imp := range imports {
			importSet[imp] = true
		}
	}

	// Convert set to slice and sort for deterministic order
	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	return imports, fileCount, nil
}

// parseFileImports parses imports from a single Go file.
func (a *Analyzer) parseFileImports(filePath string) ([]string, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseFile(a.fileSet, filePath, src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var imports []string
	for _, imp := range file.Imports {
		// Remove quotes from import path
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}

	return imports, nil
}

// getPackageName extracts a short name from a package path.
func (a *Analyzer) getPackageName(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1]
}

// buildReverseDependencyMap creates a map of what depends on each package.
func (a *Analyzer) buildReverseDependencyMap(
	graph *DependencyGraph,
	circularEdges map[string]map[string]bool,
) map[string][]string {
	reverseDeps := make(map[string][]string)

	for pkgPath, pkg := range graph.Packages {
		for _, dep := range pkg.Dependencies {
			if _, exists := graph.Packages[dep]; exists {
				// Skip circular dependencies
				if circularEdges[pkgPath] != nil && circularEdges[pkgPath][dep] {
					continue
				}
				reverseDeps[dep] = append(reverseDeps[dep], pkgPath)
			}
		}
	}

	return reverseDeps
}

// initializeLayerMap initializes all packages to unassigned layer (-1).
func initializeLayerMap(graph *DependencyGraph) map[string]int {
	layers := make(map[string]int)
	for pkgPath := range graph.Packages {
		layers[pkgPath] = -1
	}
	return layers
}

// iterateLayerCalculation performs one iteration of layer calculation.
func (a *Analyzer) iterateLayerCalculation(
	graph *DependencyGraph,
	layers map[string]int,
	reverseDeps map[string][]string,
) bool {
	changed := false

	// Process all packages in deterministic order
	packagePaths := make([]string, 0, len(graph.Packages))
	for pkgPath := range graph.Packages {
		packagePaths = append(packagePaths, pkgPath)
	}
	sort.Strings(packagePaths)

	for _, pkgPath := range packagePaths {
		newLayer := a.calculateOptimalLayer(pkgPath, layers, reverseDeps, graph)
		if layers[pkgPath] != newLayer {
			layers[pkgPath] = newLayer
			if pkg := graph.Packages[pkgPath]; pkg != nil {
				pkg.Layer = newLayer
			}
			changed = true
		}
	}

	return changed
}

// calculateOptimalLayer calculates the optimal layer for a package based on its reverse dependencies.
func (a *Analyzer) calculateOptimalLayer(
	pkgPath string,
	layers map[string]int,
	reverseDeps map[string][]string,
	graph *DependencyGraph,
) int {
	// If this package has dependents, it should be positioned above them
	maxDependentLayer := -1
	hasDependents := false
	hasCalculatedDependents := false

	for _, dependent := range reverseDeps[pkgPath] {
		if _, exists := graph.Packages[dependent]; exists {
			hasDependents = true
			if dependentLayer, calculated := layers[dependent]; calculated && dependentLayer >= 0 {
				hasCalculatedDependents = true
				if dependentLayer > maxDependentLayer {
					maxDependentLayer = dependentLayer
				}
			}
		}
	}

	if hasDependents && hasCalculatedDependents {
		// Position this package one layer above its highest dependent
		return maxDependentLayer + 1
	} else if hasDependents {
		// Has dependents but they're not calculated yet - return current layer if set, otherwise default
		if currentLayer, exists := layers[pkgPath]; exists && currentLayer >= 0 {
			return currentLayer
		}
		// Default positioning for packages with uncalculated dependents
		return 1
	}
	// True leaf package with no dependents - assign to bottom layer
	return 0
}

// organizePackagesByLayer organizes packages into layers and sorts them.
func organizePackagesByLayer(graph *DependencyGraph, layers map[string]int) {
	// Find maximum layer
	maxLayer := 0
	for _, layer := range layers {
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	// Initialize layers slice
	graph.Layers = make([][]string, maxLayer+1)

	// Assign packages to layers
	for pkgPath, layer := range layers {
		if graph.Packages[pkgPath] != nil {
			graph.Layers[layer] = append(graph.Layers[layer], pkgPath)
		}
	}

	// Sort packages within each layer for deterministic order
	for i := range graph.Layers {
		sort.Strings(graph.Layers[i])
	}
}

func (a *Analyzer) calculateLayers(graph *DependencyGraph) {
	// First, detect circular dependencies to exclude them from layer calculation
	circularEdges := a.detectCircularDependencies(graph)

	// Build reverse dependency map to understand what depends on each package
	reverseDeps := a.buildReverseDependencyMap(graph, circularEdges)

	// Initialize all packages to unassigned (-1)
	layers := initializeLayerMap(graph)

	// Use multiple passes to ensure convergence
	maxIterations := len(graph.Packages) + maxIterationsPadding
	for range maxIterations {
		if !a.iterateLayerCalculation(graph, layers, reverseDeps) {
			break // No changes occurred, we've converged
		}
	}

	// Organize packages by layer
	organizePackagesByLayer(graph, layers)
}

// detectCircularDependencies identifies packages that have circular dependencies.
func (a *Analyzer) detectCircularDependencies(graph *DependencyGraph) map[string]map[string]bool {
	circularEdges := make(map[string]map[string]bool)

	// Find all cycles using DFS
	cycles := a.findAllCycles(graph)

	// Mark all edges that are part of any cycle as circular
	for _, cycle := range cycles {
		for i := range cycle {
			from := cycle[i]
			to := cycle[(i+1)%len(cycle)]

			if circularEdges[from] == nil {
				circularEdges[from] = make(map[string]bool)
			}
			circularEdges[from][to] = true
		}
	}

	return circularEdges
}

// findAllCycles finds all cycles in the dependency graph using DFS.
func (a *Analyzer) findAllCycles(graph *DependencyGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// Try to find cycles starting from each unvisited node
	for pkgPath := range graph.Packages {
		if !visited[pkgPath] {
			path := []string{}
			a.dfsForCycles(graph, pkgPath, visited, recStack, path, &cycles)
		}
	}

	return cycles
}

// dfsForCycles performs DFS to find cycles.
func (a *Analyzer) dfsForCycles(
	graph *DependencyGraph,
	node string,
	visited, recStack map[string]bool,
	path []string,
	cycles *[][]string,
) {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	if pkg, exists := graph.Packages[node]; exists {
		a.processDependenciesForCycles(pkg, graph, visited, recStack, path, cycles)
	}

	recStack[node] = false
}

// processDependenciesForCycles processes package dependencies for cycle detection.
func (a *Analyzer) processDependenciesForCycles(
	pkg *PackageInfo,
	graph *DependencyGraph,
	visited, recStack map[string]bool,
	path []string,
	cycles *[][]string,
) {
	for _, dep := range pkg.Dependencies {
		if _, depExists := graph.Packages[dep]; !depExists {
			continue
		}

		if !visited[dep] {
			a.dfsForCycles(graph, dep, visited, recStack, path, cycles)
		} else if recStack[dep] {
			a.extractCycleFromPath(dep, path, cycles)
		}
	}
}

// extractCycleFromPath extracts a cycle from the current path.
func (a *Analyzer) extractCycleFromPath(dep string, path []string, cycles *[][]string) {
	cycleStart := -1
	for i, pathNode := range path {
		if pathNode == dep {
			cycleStart = i
			break
		}
	}
	if cycleStart != -1 {
		cycle := make([]string, len(path)-cycleStart)
		copy(cycle, path[cycleStart:])
		*cycles = append(*cycles, cycle)
	}
}

// FindEntryPoints scans a directory tree for Go files containing main functions.
func (a *Analyzer) FindEntryPoints(repoRoot string) ([]string, error) {
	var entryPoints []string

	// Convert to absolute path for consistent path handling
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving repository root: %w", err)
	}

	err = filepath.Walk(absRepoRoot, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor and .git directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
			return nil
		}

		// Check if this file contains a main function
		hasMain, err := a.fileContainsMainFunction(path)
		if err != nil {
			// Log warning but continue processing other files
			slog.Warn("Warning: failed to parse", "path", path, "error", err)
			return nil
		}

		if hasMain {
			entryPoints = append(entryPoints, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory tree: %w", err)
	}

	return entryPoints, nil
}

// fileContainsMainFunction checks if a Go file contains a main function.
func (a *Analyzer) fileContainsMainFunction(filePath string) (bool, error) {
	// Parse the file
	src, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer src.Close()

	// Parse the Go source file
	file, err := parser.ParseFile(a.fileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return false, err
	}

	// Check if this file contains a main function
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name != nil && funcDecl.Name.Name == "main" {
				// Ensure it's a function without receiver (not a method)
				if funcDecl.Recv == nil {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// validateRepositoryRoot validates the repository root path.
func validateRepositoryRoot(repoRoot string) (*MultiEntryAnalysisResult, string) {
	// Convert to absolute path
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return &MultiEntryAnalysisResult{
			Success: false,
			Error:   fmt.Sprintf("Error resolving repository path: %v", err),
		}, ""
	}

	// Check if repository root exists
	if _, statErr := os.Stat(absRepoRoot); os.IsNotExist(statErr) {
		return &MultiEntryAnalysisResult{
			Success: false,
			Error:   fmt.Sprintf("Repository root does not exist: %s", absRepoRoot),
		}, ""
	}

	return nil, absRepoRoot
}

// processEntryPoint processes a single entry point and returns an EntryPoint struct.
func (a *Analyzer) processEntryPoint(
	entryPath, absRepoRoot string,
	excludeExternal bool,
	excludeDirs []string,
) *EntryPoint {
	// Get relative path from repository root
	relPath, relErr := filepath.Rel(absRepoRoot, entryPath)
	if relErr != nil {
		slog.Warn("Warning: failed to get relative path for", "entryPath", entryPath, "error", relErr)
		return nil
	}

	// Analyze this entry point
	graph, analyzeErr := a.AnalyzeFromFile(entryPath, excludeExternal, excludeDirs)
	if analyzeErr != nil {
		slog.Warn("Warning: failed to analyze entry point", "entryPath", entryPath, "error", analyzeErr)
		return nil
	}

	// Get package path for this entry point
	pkgPath, pkgErr := a.getPackageFromFile(entryPath)
	if pkgErr != nil {
		slog.Warn("Warning: failed to get package path for",
			"entryPath", entryPath,
			"error", pkgErr)
		return nil
	}

	// Create entry point record (DOT content will be generated later)
	return &EntryPoint{
		Path:         entryPath,
		RelativePath: relPath,
		PackagePath:  pkgPath,
		DOTContent:   "", // Will be populated by the caller
		Graph:        graph,
	}
}

// processAllEntryPoints processes all entry points and returns a slice of valid EntryPoint structs.
func (a *Analyzer) processAllEntryPoints(
	entryPointPaths []string,
	absRepoRoot string,
	excludeExternal bool,
	excludeDirs []string,
) []EntryPoint {
	var entryPoints []EntryPoint

	for _, entryPath := range entryPointPaths {
		if entryPoint := a.processEntryPoint(entryPath, absRepoRoot, excludeExternal, excludeDirs); entryPoint != nil {
			entryPoints = append(entryPoints, *entryPoint)
		}
	}

	return entryPoints
}

// determineResultModuleName determines the appropriate module name for the result.
func determineResultModuleName(entryPoints []EntryPoint, absRepoRoot string) string {
	if len(entryPoints) == 0 {
		return filepath.Base(absRepoRoot)
	}

	firstModuleName := entryPoints[0].Graph.ModuleName
	allSameModule := true
	for _, ep := range entryPoints[1:] {
		if ep.Graph.ModuleName != firstModuleName {
			allSameModule = false
			break
		}
	}

	if allSameModule {
		// All entry points belong to the same module
		return firstModuleName
	}
	// Multiple modules detected (monorepo), use repository name
	return filepath.Base(absRepoRoot)
}

// AnalyzeMultipleEntryPoints finds and analyzes all entry points in a repository.
func (a *Analyzer) AnalyzeMultipleEntryPoints(
	repoRoot string,
	excludeExternal bool,
	excludeDirs []string,
) (*MultiEntryAnalysisResult, error) {
	// Validate repository root
	result, absRepoRoot := validateRepositoryRoot(repoRoot)
	if result != nil {
		return result, nil
	}
	repoRoot = absRepoRoot

	// Find all entry points
	entryPointPaths, err := a.FindEntryPoints(repoRoot)
	if err != nil {
		return &MultiEntryAnalysisResult{
			Success: false,
			Error:   fmt.Sprintf("Error finding entry points: %v", err),
		}, nil
	}

	if len(entryPointPaths) == 0 {
		return &MultiEntryAnalysisResult{
			Success: false,
			Error:   "No entry points found (files with main function)",
		}, nil
	}

	// Process all entry points
	entryPoints := a.processAllEntryPoints(entryPointPaths, repoRoot, excludeExternal, excludeDirs)

	if len(entryPoints) == 0 {
		return &MultiEntryAnalysisResult{
			Success: false,
			Error:   "No entry points could be successfully analyzed",
		}, nil
	}

	// Determine module name
	resultModuleName := determineResultModuleName(entryPoints, repoRoot)

	return &MultiEntryAnalysisResult{
		Success:     true,
		EntryPoints: entryPoints,
		RepoRoot:    repoRoot,
		ModuleName:  resultModuleName,
	}, nil
}
