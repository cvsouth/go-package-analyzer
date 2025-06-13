package scanner_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cvsouth/go-package-analyzer/internal/scanner"
)

// FuzzListDirectory tests the ListDirectory function with various directory paths.
func FuzzListDirectory(f *testing.F) {
	// Add seed corpus with various directory path formats
	f.Add(".")
	f.Add("/tmp")
	f.Add("relative/path")
	f.Add("")

	s := scanner.New()
	f.Fuzz(func(t *testing.T, dirPath string) {
		// Test that ListDirectory doesn't panic with any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ListDirectory panicked with dirPath=%q: %v", dirPath, r)
			}
		}()

		// Skip problematic inputs
		if strings.Contains(dirPath, "\x00") || len(dirPath) > 500 {
			return
		}

		result, err := s.ListDirectory(dirPath)

		// Function should never return nil result
		if result == nil {
			t.Errorf("ListDirectory returned nil result for dirPath=%q", dirPath)
			return
		}

		validateListDirectoryResult(t, result, err, dirPath)
	})
}

// validateListDirectoryResult validates the result of ListDirectory.
func validateListDirectoryResult(t *testing.T, result *scanner.DirectoryListResult, err error, dirPath string) {
	t.Helper()

	// If there's an error, should be reflected in result
	if err != nil && result.Success {
		t.Errorf("ListDirectory returned error but Success=true: dirPath=%q, error=%v", dirPath, err)
	}

	// If operation failed, should have an error message
	if !result.Success && result.Error == "" {
		t.Errorf("ListDirectory failed but no error message: dirPath=%q", dirPath)
	}

	// If successful, directories should be valid
	if result.Success {
		validateDirectoryEntries(t, result.Directories, dirPath)
	}
}

// validateDirectoryEntries validates individual directory entries.
func validateDirectoryEntries(t *testing.T, directories []*scanner.DirectoryNode, dirPath string) {
	t.Helper()

	for _, dir := range directories {
		// Each directory should have a name
		if dir.Name == "" {
			t.Errorf("Directory entry has empty name: dirPath=%q", dirPath)
		}

		// Path should be absolute or relative but not empty
		if dir.Path == "" {
			t.Errorf("Directory entry has empty path: dirPath=%q, name=%q", dirPath, dir.Name)
		}
	}
}

// FuzzGetFilesystemRoots tests filesystem root detection.
func FuzzGetFilesystemRoots(f *testing.F) {
	s := scanner.New()

	// This function doesn't take parameters, but we can test it for stability
	f.Fuzz(func(t *testing.T, _ uint8) { // dummy parameter to enable fuzzing
		// Test that GetFilesystemRoots doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetFilesystemRoots panicked: %v", r)
			}
		}()

		result, err := s.GetFilesystemRoots()

		// Function should handle errors gracefully
		if err != nil {
			return
		}

		// Result should not be nil
		if result == nil {
			t.Errorf("GetFilesystemRoots returned nil result")
			return
		}

		if result.Tree == nil {
			t.Errorf("GetFilesystemRoots returned nil tree")
			return
		}

		// Validate filesystem roots
		validateFilesystemRoots(t, result.Tree.Children)
	})
}

// validateFilesystemRoots validates filesystem root entries.
func validateFilesystemRoots(t *testing.T, roots []*scanner.DirectoryNode) {
	t.Helper()

	// Each root should be a valid path
	for i, root := range roots {
		if root.Name == "" {
			t.Errorf("Filesystem root %d has empty name", i)
		}

		// Roots should not contain relative path elements
		if strings.Contains(root.Path, "..") || strings.Contains(root.Path, "./") {
			t.Errorf("Filesystem root should not contain relative elements: %q", root.Path)
		}
	}
}

// FuzzDirectoryPathHandling tests directory path normalization and validation.
func FuzzDirectoryPathHandling(f *testing.F) {
	// Add seed corpus with various path formats
	f.Add("/")
	f.Add(".")
	f.Add("..")
	f.Add("relative/path")
	f.Add("///multiple///slashes")
	f.Add("path/with spaces")

	f.Fuzz(func(t *testing.T, dirPath string) {
		// Skip extremely problematic inputs
		if strings.Contains(dirPath, "\x00") ||
			len(dirPath) > 1000 {
			return
		}

		// Test path cleaning behavior
		cleanPath := filepath.Clean(dirPath)

		// Cleaned path should not be excessively longer than original
		// Note: filepath.Clean("") returns "." which is expected behavior
		if dirPath != "" && len(cleanPath) > len(dirPath)*3 {
			t.Errorf("Cleaned path unexpectedly long: original=%q, cleaned=%q", dirPath, cleanPath)
		}

		// Test that path operations don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Path operation panicked with dirPath=%q: %v", dirPath, r)
			}
		}()

		_ = filepath.IsAbs(dirPath)
		_ = filepath.Dir(dirPath)
	})
}
