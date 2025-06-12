package handler

import (
	"fmt"
)

// Handle processes requests
func Handle() {
	fmt.Println("Handling request...")
}

// GetServiceInfo creates a circular dependency
func GetServiceInfo() string {
	// This would normally import service, creating a cycle
	// but we'll avoid actual circular imports for valid Go code
	return "service info"
}
