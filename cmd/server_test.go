package main_test

import (
	"encoding/json"
	"testing"
)

// Test the JSON serialization of response types that would be used by the server.
type APIResponse struct {
	Success bool   `json:"success"`
	DOT     string `json:"dot,omitempty"`
	Error   string `json:"error,omitempty"`
}

type MultiEntryAPIResponse struct {
	Success    bool              `json:"success"`
	RepoRoot   string            `json:"repoRoot,omitempty"`
	ModuleName string            `json:"moduleName,omitempty"`
	Graphs     map[string]string `json:"graphs,omitempty"`
	Error      string            `json:"error,omitempty"`
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
	// Test multi-entry response
	response := MultiEntryAPIResponse{
		Success:    true,
		RepoRoot:   "/test/repo",
		ModuleName: "test/module",
		Graphs: map[string]string{
			"app1": "digraph app1 { a -> b; }",
			"app2": "digraph app2 { c -> d; }",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal MultiEntryAPIResponse: %v", err)
	}

	var unmarshaled MultiEntryAPIResponse
	if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal MultiEntryAPIResponse: %v", unmarshalErr)
	}

	if unmarshaled.Success != response.Success {
		t.Errorf("Expected Success=%v, got %v", response.Success, unmarshaled.Success)
	}

	if unmarshaled.RepoRoot != response.RepoRoot {
		t.Errorf("Expected RepoRoot='%s', got '%s'", response.RepoRoot, unmarshaled.RepoRoot)
	}

	if len(unmarshaled.Graphs) != len(response.Graphs) {
		t.Errorf("Expected %d graphs, got %d", len(response.Graphs), len(unmarshaled.Graphs))
	}

	for key, value := range response.Graphs {
		if unmarshaled.Graphs[key] != value {
			t.Errorf("Expected graph[%s]='%s', got '%s'", key, value, unmarshaled.Graphs[key])
		}
	}
}
