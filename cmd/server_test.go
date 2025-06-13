package main_test

import (
	"encoding/json"
	"testing"

	"github.com/cvsouth/go-package-analyzer/internal/analyzer"
)

// Test the JSON serialization of response types that would be used by the server.
type APIResponse struct {
	Success bool   `json:"success"`
	DOT     string `json:"dot,omitempty"`
	Error   string `json:"error,omitempty"`
}

type MultiEntryAPIResponse struct {
	Success     bool                  `json:"success"`
	EntryPoints []analyzer.EntryPoint `json:"entryPoints,omitempty"`
	Error       string                `json:"error,omitempty"`
	RepoRoot    string                `json:"repoRoot,omitempty"`
	ModuleName  string                `json:"moduleName,omitempty"`
}

func TestAPIResponse_JSONSerialization(t *testing.T) {
	// Test successful response
	response := APIResponse{
		Success: true,
		DOT:     "digraph test { a -> b; }",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal APIResponse: %v", err)
	}

	var unmarshaled APIResponse
	if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal APIResponse: %v", unmarshalErr)
	}

	if unmarshaled.Success != response.Success {
		t.Errorf("Expected Success=%v, got %v", response.Success, unmarshaled.Success)
	}

	if unmarshaled.DOT != response.DOT {
		t.Errorf("Expected DOT='%s', got '%s'", response.DOT, unmarshaled.DOT)
	}
}

func TestAPIResponse_ErrorSerialization(t *testing.T) {
	// Test error response
	response := APIResponse{
		Success: false,
		Error:   "test error message",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal APIResponse: %v", err)
	}

	var unmarshaled APIResponse
	if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal APIResponse: %v", unmarshalErr)
	}

	if unmarshaled.Success != false {
		t.Error("Expected Success=false for error response")
	}

	if unmarshaled.Error != "test error message" {
		t.Errorf("Expected Error='test error message', got '%s'", unmarshaled.Error)
	}

	if unmarshaled.DOT != "" {
		t.Errorf("Expected empty DOT for error response, got '%s'", unmarshaled.DOT)
	}
}

func TestMultiEntryAPIResponse_JSONSerialization(t *testing.T) {
	response := MultiEntryAPIResponse{
		Success:     true,
		EntryPoints: []analyzer.EntryPoint{},
		RepoRoot:    "/path/to/repo",
		ModuleName:  "test-module",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled MultiEntryAPIResponse
	if unmarshalErr := json.Unmarshal(jsonData, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
	}

	if unmarshaled.Success != response.Success {
		t.Errorf("Success field mismatch: got %v, want %v", unmarshaled.Success, response.Success)
	}

	if unmarshaled.RepoRoot != response.RepoRoot {
		t.Errorf("RepoRoot field mismatch: got %v, want %v", unmarshaled.RepoRoot, response.RepoRoot)
	}

	if unmarshaled.ModuleName != response.ModuleName {
		t.Errorf("ModuleName field mismatch: got %v, want %v", unmarshaled.ModuleName, response.ModuleName)
	}

	if len(unmarshaled.EntryPoints) != len(response.EntryPoints) {
		t.Errorf("EntryPoints length mismatch: got %d, want %d",
			len(unmarshaled.EntryPoints), len(response.EntryPoints))
	}
}
