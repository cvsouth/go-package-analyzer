// Package visualizer provides functionality for generating DOT format visualization of dependency graphs.
package visualizer

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cvsouth/go-package-analyzer/internal/analyzer"
)

// Constants for text formatting and color handling.
const (
	fillColorOpacity = 0.05 // Opacity for package fill colors
	textWrapWidth    = 25   // Maximum width for text wrapping
	hexColorLength   = 6    // Standard hex color length (RRGGBB)
)

// Visualizer generates DOT representations of package dependency graphs.
type Visualizer struct{}

// New creates a new visualizer.
func New() *Visualizer {
	return &Visualizer{}
}

// GenerateDOTContent creates DOT format content for Graphviz.
func (v *Visualizer) GenerateDOTContent(
	graph *analyzer.DependencyGraph,
) string {
	var dot strings.Builder

	v.writeDOTHeader(&dot)

	// Prepare data for node and edge generation
	packagePaths := v.getSortedPackagePaths(graph)
	circularDependencies := v.detectCircularDependencies(graph)
	dependencyPaths := v.initializeDependencyPaths(graph)

	// Generate nodes and edges
	nodeLines := v.generateNodes(graph, packagePaths, dependencyPaths)
	normalEdges, circularEdges := v.generateEdges(graph, packagePaths, circularDependencies, dependencyPaths)

	// Write output
	v.writeNodes(&dot, nodeLines)
	v.writeEdges(&dot, normalEdges, circularEdges)
	v.writeLayerConstraints(&dot, graph)

	dot.WriteString("}\n")
	return dot.String()
}

// writeDOTHeader writes the DOT file header and configuration.
func (v *Visualizer) writeDOTHeader(dot *strings.Builder) {
	dot.WriteString("digraph dependencies {\n")
	dot.WriteString("  bgcolor=\"transparent\";\n")
	dot.WriteString("  rankdir=TB;\n")
	dot.WriteString("  splines=ortho;\n")
	dot.WriteString("  nodesep=1.0;\n") // Increased from 0.8
	dot.WriteString("  ranksep=1.5;\n") // Increased from 1.2
	dot.WriteString("  concentrate=true;\n")
	dot.WriteString("  start=42;\n")           // Fixed seed for deterministic layout
	dot.WriteString("  ordering=out;\n")       // Consistent edge ordering
	dot.WriteString("  overlap=false;\n")      // Prevent node overlap
	dot.WriteString("  sep=\"+30,30\";\n")     // Increased separation
	dot.WriteString("  esep=\"+15,15\";\n")    // Increased edge separation
	dot.WriteString("  dpi=96;\n")             // Fixed DPI for consistent sizing
	dot.WriteString("  margin=\"1,1\";\n")     // Increased margin to prevent cropping
	dot.WriteString("  pad=\"1,1\";\n")        // Increased padding around the graph
	dot.WriteString("  packmode=\"graph\";\n") // Better packing to prevent overflow
	dot.WriteString(
		"  node [shape=box, style=filled, fontname=\"JetBrains Mono\", fontsize=11, penwidth=2, margin=\"0.4,0.3\", width=0, height=0, fixedsize=false];\n",
	)
	dot.WriteString("  edge [fontsize=10, labelangle=0, labeldistance=1.5];\n")
	dot.WriteString("  \n")
}

// getSortedPackagePaths returns a sorted slice of package paths for deterministic processing.
func (v *Visualizer) getSortedPackagePaths(graph *analyzer.DependencyGraph) []string {
	var packagePaths []string
	for pkgPath := range graph.Packages {
		packagePaths = append(packagePaths, pkgPath)
	}
	sort.Strings(packagePaths)
	return packagePaths
}

// initializeDependencyPaths sets up the dependency path tracking with entry point.
func (v *Visualizer) initializeDependencyPaths(graph *analyzer.DependencyGraph) map[string]int {
	dependencyPaths := make(map[string]int)
	// Entry point gets violet (first color)
	entryDepPath := v.getDependencyPath(graph.EntryPackage, graph.ModuleName)
	dependencyPaths[entryDepPath] = 0
	return dependencyPaths
}

// generateNodes creates all node definitions for the DOT output.
func (v *Visualizer) generateNodes(
	graph *analyzer.DependencyGraph,
	packagePaths []string,
	dependencyPaths map[string]int,
) []string {
	var nodeLines []string

	for _, pkgPath := range packagePaths {
		pkg := graph.Packages[pkgPath]
		nodeID := v.sanitizeNodeID(pkgPath)

		// Determine border color based on dependency path
		borderColor := v.getPackageColors(pkgPath, graph.ModuleName, dependencyPaths)

		// Create fill color as 5% opacity version of border color
		fillColor := v.hexToRGBA(borderColor, fillColorOpacity)

		// Create simple label with package name, file count, and path
		relativePath := v.getRelativePath(pkgPath, graph.ModuleName)
		wrappedPath := v.wrapText(relativePath, textWrapWidth) // Wrap path at 25 characters
		wrappedName := v.wrapText(pkg.Name, textWrapWidth)     // Wrap package name at 25 characters
		label := fmt.Sprintf("%s\\n%d files\\n%s",
			v.escapeHTML(wrappedName),
			pkg.FileCount,
			v.escapeHTML(wrappedPath))

		nodeLine := fmt.Sprintf("  %s [label=\"%s\", fillcolor=\"%s\", color=\"%s\", fontcolor=\"white\"];",
			nodeID, label, fillColor, borderColor)
		nodeLines = append(nodeLines, nodeLine)
	}

	return nodeLines
}

// generateEdges creates all edge definitions, separating normal and circular dependencies.
func (v *Visualizer) generateEdges(
	graph *analyzer.DependencyGraph,
	packagePaths []string,
	circularDependencies map[string]map[string]bool,
	dependencyPaths map[string]int,
) ([]string, []string) {
	var normalEdgeLines []string
	var circularEdgeLines []string

	for _, pkgPath := range packagePaths {
		pkg := graph.Packages[pkgPath]
		fromID := v.sanitizeNodeID(pkgPath)
		sourceBorderColor := v.getPackageColors(pkgPath, graph.ModuleName, dependencyPaths)

		// Sort dependencies for consistent edge ordering
		deps := v.getSortedDependencies(pkg, graph)

		for _, dep := range deps {
			toID := v.sanitizeNodeID(dep)

			if circularDependencies[pkgPath][dep] {
				edgeLine := v.createCircularEdge(fromID, toID, circularDependencies, pkgPath, dep)
				circularEdgeLines = append(circularEdgeLines, edgeLine)
			} else {
				edgeLine := v.createNormalEdge(fromID, toID, sourceBorderColor)
				normalEdgeLines = append(normalEdgeLines, edgeLine)
			}
		}
	}

	// Sort both edge lists for completely deterministic output
	sort.Strings(normalEdgeLines)
	sort.Strings(circularEdgeLines)

	return normalEdgeLines, circularEdgeLines
}

// getSortedDependencies returns sorted dependencies for a package.
func (v *Visualizer) getSortedDependencies(pkg *analyzer.PackageInfo, graph *analyzer.DependencyGraph) []string {
	var deps []string
	for _, dep := range pkg.Dependencies {
		if _, exists := graph.Packages[dep]; exists {
			deps = append(deps, dep)
		}
	}
	sort.Strings(deps)
	return deps
}

// createCircularEdge creates a circular dependency edge with appropriate styling.
func (v *Visualizer) createCircularEdge(
	fromID, toID string,
	circularDependencies map[string]map[string]bool,
	pkgPath, dep string,
) string {
	edgeDirection := ""
	// Check if this is a bidirectional dependency (both directions exist)
	if circularDependencies[dep] != nil && circularDependencies[dep][pkgPath] {
		edgeDirection = ", dir=both"
	}
	return fmt.Sprintf("  %s -> %s [color=\"red\", penwidth=1.5%s];", fromID, toID, edgeDirection)
}

// createNormalEdge creates a normal dependency edge.
func (v *Visualizer) createNormalEdge(fromID, toID, sourceBorderColor string) string {
	return fmt.Sprintf("  %s -> %s [color=\"%s\", penwidth=1.5];", fromID, toID, sourceBorderColor)
}

// writeNodes writes all node definitions to the DOT output.
func (v *Visualizer) writeNodes(dot *strings.Builder, nodeLines []string) {
	for _, line := range nodeLines {
		dot.WriteString(line + "\n")
	}
	dot.WriteString("  \n")
}

// writeEdges writes all edge definitions to the DOT output.
func (v *Visualizer) writeEdges(dot *strings.Builder, normalEdges, circularEdges []string) {
	// Output normal edges first
	for _, line := range normalEdges {
		dot.WriteString(line + "\n")
	}

	// Output circular edges last (so they appear "on top")
	for _, line := range circularEdges {
		dot.WriteString(line + "\n")
	}
}

// writeLayerConstraints writes layer constraints and entry point ranking to the DOT output.
func (v *Visualizer) writeLayerConstraints(dot *strings.Builder, graph *analyzer.DependencyGraph) {
	dot.WriteString("  \n")

	// First, set the entry package to be at the top with highest rank
	if graph.EntryPackage != "" {
		entryNodeID := v.sanitizeNodeID(graph.EntryPackage)
		fmt.Fprintf(dot, "  { rank=source; %s; }\n", entryNodeID)
	}

	// Generate rank constraints for each layer
	v.generateLayerConstraints(dot, graph)
}

// generateLayerConstraints generates rank constraints for graph layers.
func (v *Visualizer) generateLayerConstraints(dot *strings.Builder, graph *analyzer.DependencyGraph) {
	// Generate rank constraints for each layer (layers are indexed from 0 at top)
	// In Graphviz, rank=min is at the top, rank=max is at the bottom
	for layerIndex, layer := range graph.Layers {
		if len(layer) > 1 {
			v.processMultiPackageLayer(dot, layer, graph.EntryPackage)
		} else if len(layer) == 1 && layer[0] != graph.EntryPackage {
			v.processSinglePackageLayer(dot, layer[0], layerIndex, len(graph.Layers), graph)
		}
	}
}

// processMultiPackageLayer handles layers with multiple packages.
func (v *Visualizer) processMultiPackageLayer(dot *strings.Builder, layer []string, entryPackage string) {
	// Sort packages within the layer for deterministic output
	sortedLayer := make([]string, len(layer))
	copy(sortedLayer, layer)
	sort.Strings(sortedLayer)

	var layerNodes []string
	for _, pkgPath := range sortedLayer {
		// Skip the entry package since it's already set to rank=source
		if pkgPath != entryPackage {
			layerNodes = append(layerNodes, v.sanitizeNodeID(pkgPath))
		}
	}

	if len(layerNodes) > 0 {
		layerLine := fmt.Sprintf("  { rank=same; %s; }", strings.Join(layerNodes, "; "))
		dot.WriteString(layerLine + "\n")
	}
}

// processSinglePackageLayer handles layers with a single package.
func (v *Visualizer) processSinglePackageLayer(
	dot *strings.Builder,
	pkgPath string,
	layerIndex, totalLayers int,
	graph *analyzer.DependencyGraph,
) {
	nodeID := v.sanitizeNodeID(pkgPath)

	// For leaf packages (bottom layer), use rank=sink
	if layerIndex == totalLayers-1 && v.isLeafPackage(pkgPath, graph) {
		fmt.Fprintf(dot, "  { rank=sink; %s; }\n", nodeID)
	}
}

// isLeafPackage checks if a package has no internal dependencies.
func (v *Visualizer) isLeafPackage(pkgPath string, graph *analyzer.DependencyGraph) bool {
	pkg := graph.Packages[pkgPath]
	for _, dep := range pkg.Dependencies {
		if _, exists := graph.Packages[dep]; exists {
			return false
		}
	}
	return true
}

// detectCircularDependencies identifies packages that have circular dependencies.
func (v *Visualizer) detectCircularDependencies(graph *analyzer.DependencyGraph) map[string]map[string]bool {
	circularEdges := make(map[string]map[string]bool)

	// Find all cycles using DFS
	cycles := v.findAllCycles(graph)

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
func (v *Visualizer) findAllCycles(graph *analyzer.DependencyGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// Try to find cycles starting from each unvisited node
	for pkgPath := range graph.Packages {
		if !visited[pkgPath] {
			path := []string{}
			v.dfsForCycles(graph, pkgPath, visited, recStack, path, &cycles)
		}
	}

	return cycles
}

// dfsForCycles performs DFS to find cycles.
func (v *Visualizer) dfsForCycles(
	graph *analyzer.DependencyGraph,
	node string,
	visited, recStack map[string]bool,
	path []string,
	cycles *[][]string,
) {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	if pkg, exists := graph.Packages[node]; exists {
		v.processDependenciesForCycles(pkg, graph, visited, recStack, path, cycles)
	}

	recStack[node] = false
}

// processDependenciesForCycles processes package dependencies for cycle detection.
func (v *Visualizer) processDependenciesForCycles(
	pkg *analyzer.PackageInfo,
	graph *analyzer.DependencyGraph,
	visited, recStack map[string]bool,
	path []string,
	cycles *[][]string,
) {
	for _, dep := range pkg.Dependencies {
		if _, depExists := graph.Packages[dep]; !depExists {
			continue
		}

		if !visited[dep] {
			v.dfsForCycles(graph, dep, visited, recStack, path, cycles)
		} else if recStack[dep] {
			v.extractCycleFromPath(dep, path, cycles)
		}
	}
}

// extractCycleFromPath extracts a cycle from the current path.
func (v *Visualizer) extractCycleFromPath(dep string, path []string, cycles *[][]string) {
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

// sanitizeNodeID creates a valid DOT node identifier.
func (v *Visualizer) sanitizeNodeID(pkgPath string) string {
	// Replace problematic characters with underscores
	nodeID := strings.ReplaceAll(pkgPath, "/", "_")
	nodeID = strings.ReplaceAll(nodeID, "\\", "_") // Handle Windows backslashes
	nodeID = strings.ReplaceAll(nodeID, ".", "_")
	nodeID = strings.ReplaceAll(nodeID, "-", "_")

	// Ensure it starts with a letter or underscore
	if len(nodeID) > 0 && (nodeID[0] < 'a' || nodeID[0] > 'z') &&
		(nodeID[0] < 'A' || nodeID[0] > 'Z') && nodeID[0] != '_' {
		nodeID = "pkg_" + nodeID
	}

	return nodeID
}

// getRelativePath returns the path relative to the module (without the module namespace).
func (v *Visualizer) getRelativePath(pkgPath, moduleName string) string {
	// Remove module name prefix to get relative path
	relPath := strings.TrimPrefix(pkgPath, moduleName)
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = strings.TrimPrefix(relPath, "\\") // Handle Windows backslashes

	// Normalize path separators to forward slashes for display
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// If it's the root package, show a meaningful name
	if relPath == "" {
		return "/"
	}

	return relPath
}

// hexToRGBA converts a hex color to RGBA format with specified opacity.
func (v *Visualizer) hexToRGBA(hexColor string, opacity float64) string {
	// Remove # if present
	hex := strings.TrimPrefix(hexColor, "#")

	// Parse hex values
	var r, g, b int64
	if len(hex) == hexColorLength {
		r, _ = strconv.ParseInt(hex[0:2], 16, 0)
		g, _ = strconv.ParseInt(hex[2:4], 16, 0)
		b, _ = strconv.ParseInt(hex[4:6], 16, 0)
	} else {
		// Default to black if parsing fails
		r, g, b = 0, 0, 0
	}

	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r, g, b, opacity)
}

// getPackageColors returns fill and border colors for a package using dependency path coloring.
func (v *Visualizer) getPackageColors(
	pkgPath, moduleName string,
	dependencyPaths map[string]int,
) string {
	// Color series: border colors for dependency paths
	colorSeries := []string{
		"#6fdc8c", // Bright Pastel Mint
		"#6ab7ff", // Bright Sky Blue (pastel-leaning complement to Blue)
		"#c086e8", // Soft Bright Lavender (complement to Purple)
		"#ffe066", // Pastel Lemon (bright but soft Yellow)
		"#ff944d", // Warm Apricot (complement to Deep Orange)
		"#4dd0b0", // Pastel Aqua Teal
		"#ff80a5", // Bright Baby Pink (pastel tint of Pink)
		"#a98274", // Muted Rosewood (soft pastel Brown complement)
		"#a8e063", // Light Lime Pastel
		"#8c9eff", // Periwinkle Blue (softened Navy Blue)
		"#ff8aa1", // Coral Pink (lighter and pastel complement to Coral)
		"#b39ddb", // Light Lavender Indigo
		"#ff80bf", // Light Magenta Pink
	}

	// Get dependency path for this package
	depPath := v.getDependencyPath(pkgPath, moduleName)

	// Get color index for this dependency path
	colorIndex, exists := dependencyPaths[depPath]
	if !exists {
		// Assign next color in series
		colorIndex = len(dependencyPaths)
		dependencyPaths[depPath] = colorIndex
	}

	// Wrap around if we exceed the color series
	colorIndex %= len(colorSeries)

	borderColor := colorSeries[colorIndex]

	return borderColor
}

// getDependencyPath extracts the dependency path from a package path.
func (v *Visualizer) getDependencyPath(pkgPath, moduleName string) string {
	// Get the relative path from module
	relPath := strings.TrimPrefix(pkgPath, moduleName)
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = strings.TrimPrefix(relPath, "\\") // Handle Windows backslashes

	// Normalize path separators to forward slashes
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// If it's the root package
	if relPath == "" {
		return "root"
	}

	// Split path into components
	parts := strings.Split(relPath, "/")
	if len(parts) == 0 {
		return "root"
	}

	rootFolder := parts[0]

	// Special case: if root folder is "services", include the service name
	if rootFolder == "services" && len(parts) > 1 {
		return "services/" + parts[1]
	}

	// For other folders, just use the root folder
	return rootFolder
}

// escapeHTML escapes HTML characters for use in DOT HTML-like labels.
func (v *Visualizer) escapeHTML(text string) string {
	// First escape backslashes for DOT format (but preserve \n sequences)
	// We need to be careful not to double-escape line breaks
	text = strings.ReplaceAll(text, "\\", "\\\\")
	// But restore line breaks that got double-escaped
	text = strings.ReplaceAll(text, "\\\\n", "\\n")
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

// wrapText wraps text at a specified width, preferring to break at word boundaries.
func (v *Visualizer) wrapText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}

	// Split text while preserving separators
	var tokens []string
	currentToken := ""

	for i, char := range text {
		currentToken += string(char)

		// Check if this is a separator and we have content before it
		if (char == '/' || char == '-' || char == '_' || char == '.') && len(currentToken) > 1 {
			tokens = append(tokens, currentToken)
			currentToken = ""
		} else if i == len(text)-1 {
			// Last character
			tokens = append(tokens, currentToken)
		}
	}

	if len(tokens) <= 1 {
		// Single token or no separators - just break at maxWidth
		var result strings.Builder
		for i := 0; i < len(text); i += maxWidth {
			end := i + maxWidth
			if end > len(text) {
				end = len(text)
			}
			if i > 0 {
				result.WriteString("\\n")
			}
			result.WriteString(text[i:end])
		}
		return result.String()
	}

	// Multiple tokens - try to fit tokens on lines
	return v.wrapTokens(tokens, maxWidth)
}

// wrapTokens wraps multiple tokens onto lines within maxWidth.
func (v *Visualizer) wrapTokens(tokens []string, maxWidth int) string {
	var result strings.Builder
	currentLine := ""

	for _, token := range tokens {
		testLine := currentLine + token

		if len(testLine) <= maxWidth {
			currentLine = testLine
		} else {
			v.processTokenThatDoesntFit(&result, &currentLine, token, maxWidth)
		}
	}

	// Add the last line
	if currentLine != "" {
		if result.Len() > 0 {
			result.WriteString("\\n")
		}
		result.WriteString(currentLine)
	}

	return result.String()
}

// processTokenThatDoesntFit handles tokens that don't fit on the current line.
func (v *Visualizer) processTokenThatDoesntFit(
	result *strings.Builder, currentLine *string, token string, maxWidth int,
) {
	if *currentLine != "" {
		if result.Len() > 0 {
			result.WriteString("\\n")
		}
		result.WriteString(*currentLine)
		*currentLine = token
	} else {
		// Even single token is too long, force break it
		v.forceBreakLongToken(result, token, maxWidth)
		*currentLine = ""
	}
}

// forceBreakLongToken breaks a token that is longer than maxWidth.
func (v *Visualizer) forceBreakLongToken(result *strings.Builder, token string, maxWidth int) {
	if result.Len() > 0 {
		result.WriteString("\\n")
	}
	for i := 0; i < len(token); i += maxWidth {
		end := i + maxWidth
		if end > len(token) {
			end = len(token)
		}
		if i > 0 {
			result.WriteString("\\n")
		}
		result.WriteString(token[i:end])
	}
}
