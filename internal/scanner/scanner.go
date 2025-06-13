// Package scanner provides functionality to scan the filesystem for Go projects.
package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Operating system constants.
const (
	osWindows = "windows"
	osDarwin  = "darwin"
	osLinux   = "linux"
)

// Search depth constant for go.mod file recursive search.
const maxGoModSearchDepth = 3

// DirectoryNode represents a directory in the filesystem tree.
type DirectoryNode struct {
	Name        string           `json:"name"`
	Path        string           `json:"path"`
	IsGoProject bool             `json:"isGoProject"`
	Children    []*DirectoryNode `json:"children,omitempty"`
	IsExpanded  bool             `json:"isExpanded,omitempty"`
}

// ScanResult represents the result of a directory scan operation.
type ScanResult struct {
	Success bool           `json:"success"`
	Tree    *DirectoryNode `json:"tree,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// DirectoryListResult represents the result of listing a specific directory.
type DirectoryListResult struct {
	Success     bool             `json:"success"`
	Directories []*DirectoryNode `json:"directories"`
	Error       string           `json:"error,omitempty"`
}

// Scanner handles filesystem scanning operations.
type Scanner struct{}

// New creates a new Scanner instance.
func New() *Scanner {
	return &Scanner{}
}

// GetFilesystemRoots returns just the filesystem roots (/ for Unix, drives for Windows).
func (s *Scanner) GetFilesystemRoots() (*ScanResult, error) {
	rootPaths := getFilesystemRoots()

	if len(rootPaths) == 0 {
		return &ScanResult{
			Success: false,
			Error:   "No filesystem roots found",
		}, nil
	}

	// Create virtual root node
	root := &DirectoryNode{
		Name:     "Filesystem",
		Path:     "",
		Children: make([]*DirectoryNode, 0),
	}

	// Add each filesystem root as a child
	for _, rootPath := range rootPaths {
		var actualPath string
		var displayName string

		// For Unix systems, rootPaths contains directory names that need to be converted to absolute paths
		if (runtime.GOOS == osLinux || runtime.GOOS == osDarwin) && !strings.HasPrefix(rootPath, "/") {
			actualPath = "/" + rootPath
			displayName = rootPath
		} else {
			actualPath = rootPath
			displayName = rootPath
		}

		// Check if root is accessible - only include if it can be accessed and read
		if isDirectoryAccessible(actualPath) {
			isGo := isGoProject(actualPath)

			// If it's not a Go project, check if it has subdirectories or Go files
			// Skip root directories that are not Go projects, have no subdirectories, AND have no Go files
			if !isGo && !hasSubdirectories(actualPath) && !hasGoFiles(actualPath) {
				continue // Skip this root directory - it's a dead end with no useful content
			}

			child := &DirectoryNode{
				Name:        displayName,
				Path:        actualPath,
				IsGoProject: isGo,
				Children:    nil, // Will be loaded on demand
			}
			root.Children = append(root.Children, child)
		}
		// Note: We silently skip inaccessible roots and dead-end directories
	}

	return &ScanResult{
		Success: true,
		Tree:    root,
	}, nil
}

// processDirectoryEntries processes directory entries and returns valid directory nodes.
func processDirectoryEntries(dirPath string, entries []os.DirEntry) []*DirectoryNode {
	directories := make([]*DirectoryNode, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		childPath := filepath.Join(dirPath, entry.Name())
		childName := entry.Name()

		// Skip excluded directories
		if shouldExcludeDirectory(childPath, childName) {
			continue
		}

		// Only include child directories that are accessible
		if isDirectoryAccessible(childPath) {
			if shouldIncludeDirectory(childPath) {
				child := &DirectoryNode{
					Name:        childName,
					Path:        childPath,
					IsGoProject: isGoProject(childPath),
					Children:    nil, // Will be loaded on demand when expanded
				}
				directories = append(directories, child)
			}
		}
		// Note: We silently skip inaccessible subdirectories and dead-end directories
	}

	return directories
}

// shouldIncludeDirectory determines if a directory should be included in the results.
func shouldIncludeDirectory(childPath string) bool {
	isGo := isGoProject(childPath)

	// If it's a Go project, always include
	if isGo {
		return true
	}

	// If it's not a Go project, check if it has subdirectories or Go files
	// Skip directories that are not Go projects, have no subdirectories, AND have no Go files (dead ends)
	return hasSubdirectories(childPath) || hasGoFiles(childPath)
}

// validateDirectoryPath validates that the directory path exists and is accessible.
func validateDirectoryPath(dirPath string) *DirectoryListResult {
	// Check if directory should be excluded (but allow if it's a filesystem root)
	if !isFilesystemRoot(dirPath) && shouldExcludeDirectory(dirPath, filepath.Base(dirPath)) {
		return &DirectoryListResult{
			Success: false,
			Error:   "Directory is excluded from scanning",
		}
	}
	return nil
}

// ListDirectory returns the subdirectories of a specific directory path.
func (s *Scanner) ListDirectory(dirPath string) (*DirectoryListResult, error) {
	// Validate and clean the path
	dirPath = filepath.Clean(dirPath)

	// Check if directory is accessible upfront
	if !isDirectoryAccessible(dirPath) {
		return handleInaccessibleDirectory(dirPath)
	}

	// Validate directory path
	if result := validateDirectoryPath(dirPath); result != nil {
		return result, nil
	}

	// Read directory contents (we know this will work because we checked accessibility above)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// This shouldn't happen since we verified accessibility, but handle it just in case
		return &DirectoryListResult{
			Success: false,
			Error:   "Cannot read directory contents: " + err.Error(),
		}, err
	}

	// Process entries and get valid directories
	directories := processDirectoryEntries(dirPath, entries)

	return &DirectoryListResult{
		Success:     true,
		Directories: directories,
	}, nil
}

// isGoProject checks if a directory is a Go project by looking for go.mod file.
// A directory is considered a Go project if:
// 1. It contains a go.mod file directly in the directory
// OR
// 2. It contains a .git folder AND somewhere inside its recursive structure it contains a go.mod file.
func isGoProject(dirPath string) bool {
	// First check if go.mod file exists directly in this directory
	goModPath := filepath.Join(dirPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return true
	}

	// Check if we can access the directory - if not, assume it's not a Go project
	if _, err := os.Stat(dirPath); err != nil {
		return false
	}

	// If no direct go.mod, check if there's a .git folder
	gitPath := filepath.Join(dirPath, ".git")
	if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
		// .git exists, now recursively search for go.mod in subdirectories
		return hasGoModFileRecursive(dirPath, 0, maxGoModSearchDepth)
	}

	// No go.mod directly and no .git folder
	return false
}

// hasGoModFileRecursive recursively searches for go.mod files up to maxDepth levels.
func hasGoModFileRecursive(dirPath string, currentDepth, maxDepth int) bool {
	if currentDepth >= maxDepth {
		return false
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// If we can't read the directory (e.g., permission denied), return false
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip excluded directories to avoid scanning deep into dependencies
		childPath := filepath.Join(dirPath, entry.Name())
		if shouldExcludeDirectory(childPath, entry.Name()) {
			continue
		}

		// Check for go.mod in this subdirectory first
		goModPath := filepath.Join(childPath, "go.mod")
		if _, statErr := os.Stat(goModPath); statErr == nil {
			return true
		}

		// Recursively check deeper if not found - but only if we can access the directory
		if _, statErr := os.Stat(childPath); statErr == nil {
			if hasGoModFileRecursive(childPath, currentDepth+1, maxDepth) {
				return true
			}
		}
		// Note: We silently skip inaccessible directories rather than failing
	}

	return false
}

// getFilesystemRoots returns the filesystem roots based on the operating system.
func getFilesystemRoots() []string {
	switch runtime.GOOS {
	case osWindows:
		return getWindowsRoots()
	case osDarwin, osLinux:
		return getUnixRoots()
	default:
		return []string{"/"}
	}
}

// isFilesystemRoot checks if a path is a filesystem root.
func isFilesystemRoot(path string) bool {
	switch runtime.GOOS {
	case osWindows:
		// For Windows, check if it's a drive root like C:\, D:\, etc.
		roots := getWindowsRoots()
		for _, root := range roots {
			if path == root {
				return true
			}
		}
	case osLinux, osDarwin:
		// For Unix systems, only "/" is a true filesystem root
		// The directories returned by getUnixRoots() are just top-level directories to display
		if path == "/" {
			return true
		}
	}

	return false
}

// getWindowsRoots returns all available Windows drive letters.
func getWindowsRoots() []string {
	var roots []string

	// Check all possible drive letters
	for drive := 'A'; drive <= 'Z'; drive++ {
		drivePath := string(drive) + ":\\"
		if info, err := os.Stat(drivePath); err == nil && info.IsDir() {
			roots = append(roots, drivePath)
		}
	}

	// If no drives found (shouldn't happen), fallback to C:
	if len(roots) == 0 {
		roots = []string{"C:\\"}
	}

	return roots
}

// getUnixRoots returns the non-excluded directories within "/" for Linux and macOS.
func getUnixRoots() []string {
	var roots []string
	rootPath := "/"

	// Try to read the root directory
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		// If we can't read /, fallback to just "/"
		return []string{"/"}
	}

	// Process each entry in the root directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryName := entry.Name()
		entryPath := filepath.Join(rootPath, entryName)

		// Skip excluded directories - this will exclude system dirs like proc, sys, etc.
		if shouldExcludeDirectory(entryPath, entryName) {
			continue
		}

		// Check if the directory is both accessible and readable
		if isDirectoryAccessible(entryPath) {
			isGo := isGoProject(entryPath)

			// If it's not a Go project, check if it has subdirectories or Go files
			// Skip root directories that are not Go projects, have no subdirectories, AND have no Go files
			if !isGo && !hasSubdirectories(entryPath) && !hasGoFiles(entryPath) {
				continue // Skip this root directory - it's a dead end with no useful content
			}

			// Return just the directory name without the "/" prefix
			roots = append(roots, entryName)
		}
		// Note: We silently skip inaccessible directories and dead-end directories
	}

	// If no accessible directories found, fallback to "/"
	if len(roots) == 0 {
		roots = []string{"/"}
	}

	return roots
}

// shouldExcludeDirectory checks if a directory should be excluded from scanning.
func shouldExcludeDirectory(fullPath, dirName string) bool {
	// Check system directories
	if isSystemDirectory(dirName) {
		return true
	}

	// Check Go-specific directories
	if isGoSpecificDirectory(dirName) {
		return true
	}

	// Check Go module paths
	if isGoModulePath(fullPath) {
		return true
	}

	// Check development tools in specific locations
	if isDevToolInSpecificLocation(fullPath, dirName) {
		return true
	}

	// Check OS-specific exclusions
	return shouldExcludeOSDirectory(fullPath, dirName)
}

// isSystemDirectory checks if a directory is a system directory.
func isSystemDirectory(dirName string) bool {
	systemDirs := []string{
		"node_modules", ".git", ".svn", ".hg", "vendor",
		"bin", "obj", "tmp", "temp", "cache", ".cache",
		"log", "logs", ".logs", "dist", "build", "target",
		".idea", ".vscode", ".vs", "__pycache__", ".pytest_cache",
		".DS_Store",
	}

	for _, sysDir := range systemDirs {
		if strings.EqualFold(dirName, sysDir) {
			return true
		}
	}
	return false
}

// isGoSpecificDirectory checks if a directory is a Go-specific directory.
func isGoSpecificDirectory(dirName string) bool {
	goDirs := []string{
		"pkg",      // Go module cache
		"mod",      // Module cache subdirectory
		"sum",      // Module checksum cache
		"modcache", // Alternative module cache location
		"gocache",  // Build cache
	}

	for _, goDir := range goDirs {
		if strings.EqualFold(dirName, goDir) {
			return true
		}
	}
	return false
}

// isGoModulePath checks if a path is a Go module path.
func isGoModulePath(fullPath string) bool {
	return strings.Contains(fullPath, "/pkg/mod/") || strings.Contains(fullPath, "\\pkg\\mod\\")
}

// isDevToolInSpecificLocation checks if a directory is a development tool in a specific location.
func isDevToolInSpecificLocation(fullPath, dirName string) bool {
	devToolDirs := []string{
		"Code", "Code - Insiders", "Visual Studio Code", "code-server",
		"google-chrome", "chrome", "chromium", "firefox", "mozilla",
		"brave", "edge", "opera", "discord", "slack", "teams", "zoom",
		"docker", "docker-desktop", "virtualbox", "vmware", "parallels",
		"spotify", "steam", "android-studio", "intellij", "pycharm",
		"webstorm", "goland", "cursor", "Cursor",
	}

	for _, toolDir := range devToolDirs {
		if strings.EqualFold(dirName, toolDir) {
			// Check if this is a development tool directory in a specific location
			switch runtime.GOOS {
			case osWindows:
				return isWindowsDevToolLocation(fullPath)
			case osLinux, osDarwin:
				return isUnixDevToolLocation(fullPath)
			}
		}
	}
	return false
}

// isWindowsDevToolLocation checks if a path is a Windows development tool location.
func isWindowsDevToolLocation(fullPath string) bool {
	return strings.Contains(fullPath, "\\Program Files\\") ||
		strings.Contains(fullPath, "\\Program Files (x86)\\") ||
		strings.Contains(fullPath, "\\AppData\\") ||
		strings.Contains(fullPath, "\\Local Settings\\")
}

// isUnixDevToolLocation checks if a path is a Unix development tool location.
func isUnixDevToolLocation(fullPath string) bool {
	return strings.HasPrefix(fullPath, "/usr/") ||
		strings.HasPrefix(fullPath, "/opt/") ||
		strings.Contains(fullPath, "/.config/")
}

// shouldExcludeOSDirectory checks OS-specific directory exclusions.
func shouldExcludeOSDirectory(fullPath, dirName string) bool {
	switch runtime.GOOS {
	case osLinux:
		return shouldExcludeLinuxDirectory(fullPath, dirName)
	case osWindows:
		return shouldExcludeWindowsDirectory(fullPath, dirName)
	case osDarwin:
		return shouldExcludeMacDirectory(fullPath, dirName)
	}
	return false
}

// isLinuxSystemDirectory checks if a directory is a Linux system directory.
func isLinuxSystemDirectory(fullPath, dirName string) bool {
	linuxSystemDirs := []string{
		"proc", "sys", "dev", "run", "boot", "lost+found",
		"mnt", "media", "snap", "swapfile",
	}

	// Exclude system directories at root level
	if strings.HasPrefix(fullPath, "/") && strings.Count(fullPath, "/") == 1 {
		for _, sysDir := range linuxSystemDirs {
			if strings.EqualFold(dirName, sysDir) {
				return true
			}
		}
	}
	return false
}

// isBrowserOrAppDirectory checks if a directory is a browser or application data directory.
func isBrowserOrAppDirectory(fullPath, dirName string) bool {
	browserAndAppDirs := []string{
		".mozilla", ".firefox", ".chrome", ".chromium", ".google-chrome",
		".config/google-chrome", ".config/chromium", ".config/mozilla",
		".thunderbird", ".steam", ".discord", ".slack", ".zoom", ".docker",
		".android", ".gradle", ".npm", ".yarn", ".pnpm", ".cargo", ".rustup",
		".gem", ".rbenv", ".pyenv", ".virtualenvs", ".conda", ".miniconda", ".anaconda",
	}

	// Check if this is a browser/app directory (full path or name match)
	for _, appDir := range browserAndAppDirs {
		if strings.Contains(fullPath, appDir) || strings.EqualFold(dirName, filepath.Base(appDir)) {
			return true
		}
	}
	return false
}

// isConfigDevToolDirectory checks if a directory is a development tool within .config.
func isConfigDevToolDirectory(fullPath, dirName string) bool {
	if !strings.Contains(fullPath, "/.config/") {
		return false
	}

	configDevToolDirs := []string{
		"Code", "Code - Insiders", "Visual Studio Code", "code-server",
		"google-chrome", "chromium", "mozilla", "discord", "slack", "zoom",
		"docker", "android-studio", "cursor", "Cursor", "intellij",
		"pycharm", "webstorm", "goland",
	}

	for _, toolDir := range configDevToolDirs {
		if strings.EqualFold(dirName, toolDir) {
			return true
		}
	}
	return false
}

// isAllowedHiddenDirectory checks if a hidden directory is allowed.
func isAllowedHiddenDirectory(dirName string) bool {
	if !strings.HasPrefix(dirName, ".") {
		return true // Not a hidden directory
	}

	allowedHiddenDirs := []string{
		".config", ".local", ".ssh", ".gnupg", ".gitconfig", ".bashrc",
		".zshrc", ".profile", ".vimrc", ".tmux", ".aws", ".kube", ".terraform",
	}

	for _, allowedDir := range allowedHiddenDirs {
		if strings.HasPrefix(dirName, allowedDir) {
			return true
		}
	}
	return false
}

func shouldExcludeLinuxDirectory(fullPath, dirName string) bool {
	// Check system directories
	if isLinuxSystemDirectory(fullPath, dirName) {
		return true
	}

	// Check browser and application data directories
	if isBrowserOrAppDirectory(fullPath, dirName) {
		return true
	}

	// Check development tools within .config
	if isConfigDevToolDirectory(fullPath, dirName) {
		return true
	}

	// Check hidden directories (exclude most, allow some)
	if !isAllowedHiddenDirectory(dirName) {
		return true
	}

	return false
}

// shouldExcludeWindowsDirectory checks for Windows-specific directory exclusions.
func shouldExcludeWindowsDirectory(fullPath, dirName string) bool {
	windowsSystemDirs := []string{
		"Windows", "Program Files", "Program Files (x86)", "ProgramData",
		"System Volume Information", "$Recycle.Bin", "Recovery",
		"hiberfil.sys", "pagefile.sys", "swapfile.sys",
		"AppData", "Application Data", "Local Settings",
	}

	for _, sysDir := range windowsSystemDirs {
		if strings.EqualFold(dirName, sysDir) {
			return true
		}
	}

	// Exclude drive root system folders
	if len(fullPath) >= 3 && fullPath[1] == ':' && fullPath[2] == '\\' {
		if strings.Count(fullPath, "\\") == 1 {
			for _, sysDir := range windowsSystemDirs {
				if strings.EqualFold(dirName, sysDir) {
					return true
				}
			}
		}
	}

	return false
}

// shouldExcludeMacDirectory checks for macOS-specific directory exclusions.
func shouldExcludeMacDirectory(fullPath, dirName string) bool {
	macSystemDirs := []string{
		"System", "Library", "Applications", "Volumes", "cores",
		"dev", "etc", "tmp", "usr", "bin", "sbin", "var",
		"private", "Network", ".DocumentRevisions-V100",
		".Spotlight-V100", ".Trashes", ".fseventsd",
	}

	// Exclude system directories at root level
	if strings.HasPrefix(fullPath, "/") && strings.Count(fullPath, "/") == 1 {
		for _, sysDir := range macSystemDirs {
			if strings.EqualFold(dirName, sysDir) {
				return true
			}
		}
	}

	// Exclude hidden directories (but allow some common ones)
	if strings.HasPrefix(dirName, ".") &&
		!strings.HasPrefix(dirName, ".config") &&
		!strings.HasPrefix(dirName, ".local") {
		return true
	}

	return false
}

// ScanForGoProjects is a legacy method for backwards compatibility - now just returns filesystem roots.
func (s *Scanner) ScanForGoProjects() (*ScanResult, error) {
	return s.GetFilesystemRoots()
}

// isDirectoryAccessible checks if a directory exists, is accessible, and can be read.
func isDirectoryAccessible(dirPath string) bool {
	// First check if directory exists and is a directory
	info, err := os.Stat(dirPath)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Try to read the directory to ensure we have read permissions
	_, err = os.ReadDir(dirPath)
	return err == nil
}

// hasSubdirectories checks if a directory contains any subdirectories.
func hasSubdirectories(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false // If we can't read it, assume no subdirectories
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}

// hasGoFiles checks if a directory contains any .go files.
func hasGoFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false // If we can't read it, assume no Go files
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

// handleInaccessibleDirectory handles error cases when a directory is not accessible.
func handleInaccessibleDirectory(dirPath string) (*DirectoryListResult, error) {
	// Check specific error types for better error messages
	if info, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return &DirectoryListResult{
				Success: false,
				Error:   "Directory does not exist",
			}, nil
		} else if os.IsPermission(err) {
			return &DirectoryListResult{
				Success: false,
				Error:   "Permission denied - cannot access directory",
			}, nil
		}
		return &DirectoryListResult{
			Success: false,
			Error:   "Directory not accessible: " + err.Error(),
		}, nil
	} else if !info.IsDir() {
		return &DirectoryListResult{
			Success: false,
			Error:   "Path is not a directory",
		}, nil
	}
	// Directory exists but can't be read
	return &DirectoryListResult{
		Success: false,
		Error:   "Permission denied - cannot read directory contents",
	}, nil
}
