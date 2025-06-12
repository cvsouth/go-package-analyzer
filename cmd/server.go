// Package main provides the HTTP server for the Go package analyzer.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"cvsouth/go-package-analyzer/internal/analyzer"
	"cvsouth/go-package-analyzer/internal/visualizer"
)

// Server timeout constants.
const (
	serverReadTimeout       = 60 * time.Second  // HTTP server read timeout
	serverWriteTimeout      = 60 * time.Second  // HTTP server write timeout
	serverIdleTimeout       = 120 * time.Second // HTTP server idle timeout
	serverReadHeaderTimeout = 10 * time.Second  // HTTP server read header timeout
	serverMaxHeaderBytes    = 1 << 20           // HTTP server max header bytes (1 MB)
	serverShutdownTimeout   = 30 * time.Second  // Server graceful shutdown timeout
)

// APIResponse represents the response structure for the API.
type APIResponse struct {
	Success bool   `json:"success"`
	DOT     string `json:"dot,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MultiEntryAPIResponse represents the response structure for multi-entry analysis.
type MultiEntryAPIResponse struct {
	Success     bool                  `json:"success"`
	EntryPoints []analyzer.EntryPoint `json:"entryPoints,omitempty"`
	Error       string                `json:"error,omitempty"`
	RepoRoot    string                `json:"repoRoot,omitempty"`
	ModuleName  string                `json:"moduleName,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "6333"
	}

	server := &http.Server{
		Addr:              ":" + port,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./web/")))

	mux.HandleFunc("/api/analyze", handleAnalyze)
	mux.HandleFunc("/api/analyze-repo", handleAnalyzeRepo)

	server.Handler = mux

	slog.Info("Server starting on http://localhost:" + port)

	sigChan := make(chan os.Signal, 1)
	// Use only cross-platform signals that work on all systems
	signal.Notify(sigChan,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Termination request
	)

	// Start server in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("PANIC in server goroutine",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())))
				sigChan <- syscall.SIGTERM
			}
		}()

		err := server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("FATAL: HTTP server error", slog.Any("error", err))
			sigChan <- syscall.SIGTERM
		}
	}()

	<-sigChan

	// Give outstanding requests a deadline to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer shutdownCancel()

	// Shut down server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown",
			slog.Any("error", err))
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		slog.Info("handleAnalyze: Method not allowed", slog.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	entryFile := r.URL.Query().Get("entry")
	showExternalStr := r.URL.Query().Get("external")
	excludeDirsStr := r.URL.Query().Get("exclude")

	if entryFile == "" {
		sendJSONResponse(w, APIResponse{
			Success: false,
			Error:   "entry parameter is required",
		})
		return
	}

	// Convert relative path to absolute
	absEntryFile, err := filepath.Abs(entryFile)
	if err != nil {
		sendJSONResponse(w, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Error resolving entry file path: %v", err),
		})
		return
	}

	// Check if entry file exists
	if _, statErr := os.Stat(absEntryFile); os.IsNotExist(statErr) {
		sendJSONResponse(w, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Entry file does not exist: %s", absEntryFile),
		})
		return
	}

	// Parse parameters
	showExternal := showExternalStr == "true"
	var excludeList []string
	if excludeDirsStr != "" {
		excludeList = strings.Split(excludeDirsStr, ",")
		for i, dir := range excludeList {
			excludeList[i] = strings.TrimSpace(dir)
		}
	}

	// Analyze the codebase
	analyze := analyzer.New()
	graph, err := analyze.AnalyzeFromFile(absEntryFile, !showExternal, excludeList)
	if err != nil {
		slog.Error("handleAnalyze: Analysis failed", slog.Any("error", err))
		sendJSONResponse(w, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Error analyzing codebase: %v", err),
		})
		return
	}

	if len(graph.Packages) == 0 {
		sendJSONResponse(w, APIResponse{
			Success: false,
			Error:   "No packages found to analyze",
		})
		return
	}

	// Generate DOT content
	viz := visualizer.New()
	dotContent := viz.GenerateDOTContent(graph)

	sendJSONResponse(w, APIResponse{
		Success: true,
		DOT:     dotContent,
	})
}

func handleAnalyzeRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		slog.Info("handleAnalyzeRepo: Method not allowed", slog.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	repoRoot := r.URL.Query().Get("repo")
	showExternalStr := r.URL.Query().Get("external")
	excludeDirsStr := r.URL.Query().Get("exclude")

	if repoRoot == "" {
		sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
			Success: false,
			Error:   "repo parameter is required",
		})
		return
	}

	// Convert relative path to absolute
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("Error resolving repository path: %v", err),
		})
		return
	}

	// Check if repository root exists
	if _, statErr := os.Stat(absRepoRoot); os.IsNotExist(statErr) {
		sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("Repository root does not exist: %s", absRepoRoot),
		})
		return
	}

	// Parse parameters
	showExternal := showExternalStr == "true"
	var excludeList []string
	if excludeDirsStr != "" {
		excludeList = strings.Split(excludeDirsStr, ",")
		for i, dir := range excludeList {
			excludeList[i] = strings.TrimSpace(dir)
		}
	}

	// Analyze the repository
	analyze := analyzer.New()
	result, err := analyze.AnalyzeMultipleEntryPoints(absRepoRoot, !showExternal, excludeList)
	if err != nil {
		slog.Error("handleAnalyzeRepo: Repository analysis failed", slog.Any("error", err))
		sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("Error analyzing repository: %v", err),
		})
		return
	}

	if !result.Success {
		sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
			Success: false,
			Error:   result.Error,
		})
		return
	}

	// Generate DOT content for each entry point
	viz := visualizer.New()
	for i := range result.EntryPoints {
		if result.EntryPoints[i].Graph != nil {
			result.EntryPoints[i].DOTContent = viz.GenerateDOTContent(result.EntryPoints[i].Graph)
		}
	}
	sendMultiEntryJSONResponse(w, MultiEntryAPIResponse{
		Success:     true,
		EntryPoints: result.EntryPoints,
		RepoRoot:    result.RepoRoot,
		ModuleName:  result.ModuleName,
	})
}

func sendJSONResponse(w http.ResponseWriter, response APIResponse) {
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("sendJSONResponse: Error encoding response", slog.Any("error", err))
		return
	}
}

func sendMultiEntryJSONResponse(w http.ResponseWriter, response MultiEntryAPIResponse) {
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("sendMultiEntryJSONResponse: Error encoding response", slog.Any("error", err))
		return
	}
}
