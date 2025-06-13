package scanner_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cvsouth/go-package-analyzer/internal/scanner"
)

// Test helper functions and utilities

// createTempDirWithStructure creates a complex directory structure for testing.
func createTempDirWithStructure(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create various subdirectories
	dirs := []string{
		"go_project_with_mod",
		"go_project_with_git",
		"go_project_with_git/subdir",
		"regular_dir",
		"regular_dir/subdir",
		"empty_dir",
		"node_modules", // Should be excluded
		".git",         // Should be excluded
		"vendor",       // Should be excluded
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	// Create go.mod in go project
	goModContent := "module test\n\ngo 1.19\n"
	err := os.WriteFile(filepath.Join(tempDir, "go_project_with_mod", "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create .git folder and go.mod in subdirectory for git project
	gitDir := filepath.Join(tempDir, "go_project_with_git", ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "go_project_with_git", "subdir", "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create some Go files
	goFileContent := "package main\n\nfunc main() {}\n"
	err = os.WriteFile(filepath.Join(tempDir, "regular_dir", "main.go"), []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Create regular files
	err = os.WriteFile(filepath.Join(tempDir, "regular_dir", "README.md"), []byte("# Test"), 0644)
	require.NoError(t, err)

	return tempDir
}

// createRestrictedDir creates a directory with restricted permissions (Unix only).
func createRestrictedDir(t *testing.T) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not applicable on Windows")
	}

	tempDir := t.TempDir()

	// Create a subdirectory and restrict its permissions
	restrictedDir := filepath.Join(tempDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0755)
	require.NoError(t, err)

	// Remove read permissions
	err = os.Chmod(restrictedDir, 0000)
	require.NoError(t, err)

	return tempDir
}

// cleanupTempDir removes temporary directory and handles permission restoration.
func cleanupTempDir(t *testing.T, tempDir string) {
	t.Helper()

	// Restore permissions before cleanup (Unix only)
	if runtime.GOOS != "windows" {
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() {
				os.Chmod(path, 0755)
			}
			return nil
		})
	}

	err := os.RemoveAll(tempDir)
	if err != nil {
		t.Logf("Failed to cleanup temp dir %s: %v", tempDir, err)
	}
}

// Test Data Structures

func TestDirectoryNode_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		node *scanner.DirectoryNode
	}{
		{
			name: "empty node",
			node: &scanner.DirectoryNode{},
		},
		{
			name: "simple node",
			node: &scanner.DirectoryNode{
				Name:        "test",
				Path:        "/test",
				IsGoProject: true,
				IsExpanded:  true,
			},
		},
		{
			name: "node with children",
			node: &scanner.DirectoryNode{
				Name:        "parent",
				Path:        "/parent",
				IsGoProject: false,
				Children: []*scanner.DirectoryNode{
					{
						Name:        "child1",
						Path:        "/parent/child1",
						IsGoProject: true,
					},
					{
						Name:        "child2",
						Path:        "/parent/child2",
						IsGoProject: false,
					},
				},
			},
		},
		{
			name: "node with unicode characters",
			node: &scanner.DirectoryNode{
				Name:        "测试目录",
				Path:        "/测试目录",
				IsGoProject: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.node)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Test unmarshaling
			var unmarshaled scanner.DirectoryNode
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify fields
			assert.Equal(t, tt.node.Name, unmarshaled.Name)
			assert.Equal(t, tt.node.Path, unmarshaled.Path)
			assert.Equal(t, tt.node.IsGoProject, unmarshaled.IsGoProject)
			assert.Equal(t, tt.node.IsExpanded, unmarshaled.IsExpanded)
			assert.Len(t, unmarshaled.Children, len(tt.node.Children))
		})
	}
}

func TestScanResult_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		result *scanner.ScanResult
	}{
		{
			name: "success result",
			result: &scanner.ScanResult{
				Success: true,
				Tree: &scanner.DirectoryNode{
					Name: "root",
					Path: "/",
				},
			},
		},
		{
			name: "error result",
			result: &scanner.ScanResult{
				Success: false,
				Error:   "test error",
			},
		},
		{
			name:   "empty result",
			result: &scanner.ScanResult{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var unmarshaled scanner.ScanResult
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.result.Success, unmarshaled.Success)
			assert.Equal(t, tt.result.Error, unmarshaled.Error)
		})
	}
}

func TestDirectoryListResult_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		result *scanner.DirectoryListResult
	}{
		{
			name: "success result with directories",
			result: &scanner.DirectoryListResult{
				Success: true,
				Directories: []*scanner.DirectoryNode{
					{Name: "dir1", Path: "/dir1"},
					{Name: "dir2", Path: "/dir2"},
				},
			},
		},
		{
			name: "error result",
			result: &scanner.DirectoryListResult{
				Success: false,
				Error:   "test error",
			},
		},
		{
			name: "empty result",
			result: &scanner.DirectoryListResult{
				Success:     true,
				Directories: []*scanner.DirectoryNode{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var unmarshaled scanner.DirectoryListResult
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.result.Success, unmarshaled.Success)
			assert.Equal(t, tt.result.Error, unmarshaled.Error)
			assert.Len(t, unmarshaled.Directories, len(tt.result.Directories))
		})
	}
}

// Test Constructor

func TestNew(t *testing.T) {
	scanner1 := scanner.New()
	scanner2 := scanner.New()

	// Verify non-nil return
	assert.NotNil(t, scanner1)
	assert.NotNil(t, scanner2)

	// Verify they are different instances (different pointers)
	assert.NotSame(t, scanner1, scanner2, "Should create different instances")
}

// Test GetFilesystemRoots

func TestScanner_GetFilesystemRoots(t *testing.T) {
	s := scanner.New()

	result, err := s.GetFilesystemRoots()
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should always succeed
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Tree)

	// Check root node properties
	assert.Equal(t, "Filesystem", result.Tree.Name)
	assert.Empty(t, result.Tree.Path)
	assert.NotNil(t, result.Tree.Children)

	// Should have at least one child (filesystem root)
	assert.NotEmpty(t, result.Tree.Children)

	// Verify OS-specific behavior
	switch runtime.GOOS {
	case "windows":
		// Windows should have drive letters
		for _, child := range result.Tree.Children {
			assert.True(t, strings.HasSuffix(child.Path, ":\\") || strings.HasSuffix(child.Path, ":/"))
		}
	case "linux", "darwin":
		// Unix systems should have root directory or subdirectories
		foundRoot := false
		for _, child := range result.Tree.Children {
			if strings.HasPrefix(child.Path, "/") {
				foundRoot = true
				break
			}
		}
		assert.True(t, foundRoot, "Should find at least one Unix-style path")
	}
}

// Test ListDirectory

func TestScanner_ListDirectory_ValidDirectories(t *testing.T) {
	s := scanner.New()
	tempDir := createTempDirWithStructure(t)
	defer cleanupTempDir(t, tempDir)

	result, err := s.ListDirectory(tempDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Directories)

	// Find expected directories (excluding system directories)
	dirNames := make(map[string]bool)
	for _, dir := range result.Directories {
		dirNames[dir.Name] = true
	}

	// Should include Go projects
	assert.True(t, dirNames["go_project_with_mod"], "Should include directory with go.mod")
	assert.True(t, dirNames["go_project_with_git"], "Should include directory with .git and nested go.mod")

	// Should include directories with Go files
	assert.True(t, dirNames["regular_dir"], "Should include directory with .go files")

	// Should exclude system directories
	assert.False(t, dirNames["node_modules"], "Should exclude node_modules")
	assert.False(t, dirNames[".git"], "Should exclude .git")
	assert.False(t, dirNames["vendor"], "Should exclude vendor")

	// Should exclude empty directories (no Go content)
	assert.False(t, dirNames["empty_dir"], "Should exclude empty directories")
}

// setupGoProjectTestCase creates a test directory with specific Go project characteristics.
func setupGoProjectTestCase(t *testing.T, baseDir, testName string, hasGoMod, hasGitFolder bool) string {
	t.Helper()

	testDir := filepath.Join(baseDir, "test_"+strings.ReplaceAll(testName, " ", "_"))
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	if hasGoMod {
		goModContent := "module test\n\ngo 1.19\n"
		err = os.WriteFile(filepath.Join(testDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)
	}

	if hasGitFolder {
		setupGitFolder(t, testDir, !hasGoMod)
	}

	return testDir
}

// setupGitFolder creates a .git directory and optionally a nested go.mod.
func setupGitFolder(t *testing.T, testDir string, createNestedGoMod bool) {
	t.Helper()

	gitDir := filepath.Join(testDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	if createNestedGoMod {
		subDir := filepath.Join(testDir, "subdir")
		err = os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		goModContent := "module test\n\ngo 1.19\n"
		err = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)
	}
}

// validateGoProjectDetection validates that the directory was correctly detected as a Go project.
func validateGoProjectDetection(t *testing.T, s *scanner.Scanner, baseDir, testDir string, expectedGo bool) {
	t.Helper()

	result, err := s.ListDirectory(baseDir)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Find our test directory
	var foundDir *scanner.DirectoryNode
	for _, dir := range result.Directories {
		if dir.Path == testDir {
			foundDir = dir
			break
		}
	}

	if expectedGo {
		require.NotNil(t, foundDir, "Go project directory should be included")
		assert.True(t, foundDir.IsGoProject, "Directory should be detected as Go project")
	} else if foundDir != nil {
		// Non-Go directories without content may be excluded
		assert.False(t, foundDir.IsGoProject, "Directory should not be detected as Go project")
	}
}

func TestScanner_ListDirectory_GoProjectDetection(t *testing.T) {
	s := scanner.New()

	// Create a base test directory in the current working directory to avoid system exclusions
	baseDir := t.TempDir()
	defer cleanupTempDir(t, baseDir)

	tests := []struct {
		name         string
		hasGoMod     bool
		hasGitFolder bool
		expectedGo   bool
	}{
		{
			name:       "directory with go.mod",
			hasGoMod:   true,
			expectedGo: true,
		},
		{
			name:         "directory with .git and nested go.mod",
			hasGitFolder: true,
			expectedGo:   true,
		},
		{
			name:       "directory without Go project markers",
			expectedGo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := setupGoProjectTestCase(t, baseDir, tt.name, tt.hasGoMod, tt.hasGitFolder)
			validateGoProjectDetection(t, s, baseDir, testDir, tt.expectedGo)
		})
	}
}

func TestScanner_ListDirectory_ErrorCases(t *testing.T) {
	s := scanner.New()

	tests := []struct {
		name          string
		dirPath       string
		expectedError string
	}{
		{
			name:          "non-existent directory",
			dirPath:       "/this/directory/does/not/exist",
			expectedError: "Directory does not exist",
		},
		{
			name:          "empty path",
			dirPath:       "",
			expectedError: "", // Clean path might resolve to current directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.ListDirectory(tt.dirPath)
			require.NoError(t, err) // Method shouldn't return error, but result should indicate failure
			require.NotNil(t, result)

			if tt.expectedError != "" {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.expectedError)
			}
		})
	}
}

func TestScanner_ListDirectory_FileInsteadOfDirectory(t *testing.T) {
	s := scanner.New()

	// Create a temp file
	tempFile, err := os.CreateTemp(t.TempDir(), "scanner_test_file_*")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	result, err := s.ListDirectory(tempFile.Name())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not a directory")
}

func TestScanner_ListDirectory_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not reliable on Windows")
	}

	s := scanner.New()
	tempDir := createRestrictedDir(t)
	defer cleanupTempDir(t, tempDir)

	restrictedPath := filepath.Join(tempDir, "restricted")
	result, err := s.ListDirectory(restrictedPath)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Success)
	assert.Contains(t, strings.ToLower(result.Error), "permission")
}

func TestScanner_ListDirectory_PathTraversalSafety(t *testing.T) {
	s := scanner.New()

	// Test various path traversal attempts
	paths := []string{
		"../../../",
		"..\\..\\..\\",
		"/../../etc",
		"./../..",
	}

	for _, path := range paths {
		t.Run("path_"+path, func(t *testing.T) {
			result, err := s.ListDirectory(path)
			require.NoError(t, err)

			// Should either succeed with cleaned path or fail gracefully
			// The important thing is it doesn't panic or cause security issues
			assert.NotNil(t, result)
		})
	}
}

func TestScanner_ListDirectory_UnicodeAndSpecialChars(t *testing.T) {
	s := scanner.New()

	// Create temp directory with unicode name
	tempDir := t.TempDir()
	defer cleanupTempDir(t, tempDir)

	// Create subdirectory with unicode name
	unicodeSubdir := filepath.Join(tempDir, "子目录")
	err := os.MkdirAll(unicodeSubdir, 0755)
	require.NoError(t, err)

	// Create go.mod to make it a Go project
	goModContent := "module test\n\ngo 1.19\n"
	err = os.WriteFile(filepath.Join(unicodeSubdir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	result, err := s.ListDirectory(tempDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Directories)

	// Find unicode directory
	var foundUnicodeDir *scanner.DirectoryNode
	for _, dir := range result.Directories {
		if dir.Name == "子目录" {
			foundUnicodeDir = dir
			break
		}
	}

	require.NotNil(t, foundUnicodeDir, "Should find unicode directory")
	assert.True(t, foundUnicodeDir.IsGoProject, "Unicode directory should be detected as Go project")
}

// Test ScanForGoProjects (legacy method)

func TestScanner_ScanForGoProjects(t *testing.T) {
	s := scanner.New()

	result1, err1 := s.ScanForGoProjects()
	require.NoError(t, err1)

	result2, err2 := s.GetFilesystemRoots()
	require.NoError(t, err2)

	// Legacy method should return same result as GetFilesystemRoots
	assert.Equal(t, result1.Success, result2.Success)
	assert.Equal(t, result1.Error, result2.Error)

	// Tree structure should be the same
	if result1.Tree != nil && result2.Tree != nil {
		assert.Equal(t, result1.Tree.Name, result2.Tree.Name)
		assert.Equal(t, result1.Tree.Path, result2.Tree.Path)
		assert.Len(t, result2.Tree.Children, len(result1.Tree.Children))
	}
}

// Integration Tests

func TestScanner_Integration_ComplexDirectoryStructure(t *testing.T) {
	s := scanner.New()

	// Create a complex nested structure
	tempDir := t.TempDir()
	defer cleanupTempDir(t, tempDir)

	// Create various directory types
	structure := map[string][]string{
		"go_project":            {"go.mod"},
		"git_go_project":        {".git/", "src/go.mod"},
		"regular_with_go":       {"main.go", "utils.go"},
		"regular_no_go":         {"README.md", "config.json"},
		"empty":                 {},
		"node_modules":          {"package.json"}, // Should be excluded
		".hidden":               {"file.txt"},     // Should be excluded
		"nested/deep/structure": {"go.mod"},
	}

	for dirPath, files := range structure {
		fullDirPath := filepath.Join(tempDir, dirPath)
		err := os.MkdirAll(fullDirPath, 0755)
		require.NoError(t, err)

		for _, file := range files {
			filePath := filepath.Join(fullDirPath, file)
			if strings.HasSuffix(file, "/") {
				// It's a directory
				err = os.MkdirAll(filePath, 0755)
				require.NoError(t, err)
			} else {
				// Ensure parent directory exists
				err = os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)

				var content string
				switch filepath.Ext(file) {
				case ".mod":
					content = "module test\n\ngo 1.19\n"
				case ".go":
					content = "package main\n\nfunc main() {}\n"
				default:
					content = "test content"
				}

				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}
		}
	}

	result, err := s.ListDirectory(tempDir)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Analyze results
	dirMap := make(map[string]*scanner.DirectoryNode)
	for _, dir := range result.Directories {
		dirMap[dir.Name] = dir
	}

	// Verify Go project detection
	assert.True(t, dirMap["go_project"].IsGoProject, "Direct go.mod should be detected")
	assert.True(t, dirMap["git_go_project"].IsGoProject, "Git repo with nested go.mod should be detected")

	// Verify non-Go projects with Go files are included
	if regularDir, exists := dirMap["regular_with_go"]; exists {
		assert.False(t, regularDir.IsGoProject, "Directory with only .go files should not be Go project")
	}

	// Verify exclusions
	assert.NotContains(t, dirMap, "node_modules", "node_modules should be excluded")
	assert.NotContains(t, dirMap, ".hidden", "Hidden directories should be excluded")

	// Verify nested structure is accessible
	if nestedDir, exists := dirMap["nested"]; exists {
		// Test listing the nested directory
		nestedResult, nestedErr := s.ListDirectory(nestedDir.Path)
		require.NoError(t, nestedErr)
		assert.True(t, nestedResult.Success)
	}
}

func TestScanner_Integration_CrossPlatformPaths(t *testing.T) {
	s := scanner.New()

	// Test that paths are handled correctly across platforms
	tempDir := t.TempDir()
	defer cleanupTempDir(t, tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create go.mod
	goModContent := "module test\n\ngo 1.19\n"
	err = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	result, err := s.ListDirectory(tempDir)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.NotEmpty(t, result.Directories)

	subDirResult := result.Directories[0]

	// Verify path uses correct separators for the platform
	expectedPath := filepath.Join(tempDir, "subdir")
	assert.Equal(t, expectedPath, subDirResult.Path)

	// Verify path can be used for further operations
	nestedResult, err := s.ListDirectory(subDirResult.Path)
	require.NoError(t, err)
	assert.True(t, nestedResult.Success)
}

// Performance Tests

func TestScanner_Performance_LargeDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	s := scanner.New()

	// Create a directory with many subdirectories
	tempDir := t.TempDir()
	defer cleanupTempDir(t, tempDir)

	// Create 100 subdirectories (reasonable for performance test)
	for i := range 100 {
		subDir := filepath.Join(tempDir, fmt.Sprintf("dir_%03d", i))
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Every 10th directory gets a go.mod
		if i%10 == 0 {
			goModContent := fmt.Sprintf("module dir_%03d\n\ngo 1.19\n", i)
			err = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte(goModContent), 0644)
			require.NoError(t, err)
		}
	}

	// Measure performance
	start := time.Now()
	result, err := s.ListDirectory(tempDir)
	duration := time.Since(start)

	require.NoError(t, err)
	require.True(t, result.Success)

	// Should complete within reasonable time (5 seconds for 100 dirs)
	assert.Less(t, duration, 5*time.Second, "Directory listing should complete within 5 seconds")

	// Should find the Go projects
	goProjectCount := 0
	for _, dir := range result.Directories {
		if dir.IsGoProject {
			goProjectCount++
		}
	}
	assert.Equal(t, 10, goProjectCount, "Should find exactly 10 Go projects")
}

// Error Recovery Tests

func TestScanner_ErrorRecovery_MixedPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not reliable on Windows")
	}

	s := scanner.New()

	tempDir := t.TempDir()
	defer cleanupTempDir(t, tempDir)

	// Create mixed accessible/inaccessible directories
	accessibleDir := filepath.Join(tempDir, "accessible")
	err := os.MkdirAll(accessibleDir, 0755)
	require.NoError(t, err)

	restrictedDir := filepath.Join(tempDir, "restricted")
	err = os.MkdirAll(restrictedDir, 0755)
	require.NoError(t, err)

	// Add go.mod to accessible directory
	goModContent := "module accessible\n\ngo 1.19\n"
	err = os.WriteFile(filepath.Join(accessibleDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Restrict permissions on one directory
	err = os.Chmod(restrictedDir, 0000)
	require.NoError(t, err)

	result, err := s.ListDirectory(tempDir)
	require.NoError(t, err)
	require.True(t, result.Success) // Should succeed despite restricted directory

	// Should still find accessible directory
	found := false
	for _, dir := range result.Directories {
		if dir.Name == "accessible" {
			found = true
			assert.True(t, dir.IsGoProject)
			break
		}
	}
	assert.True(t, found, "Should find accessible directory despite restricted directory")
}
