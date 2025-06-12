package service

import (
	"fmt"
	"testing/data/complex_project/handler"
)

// Start initializes the service
func Start() {
	fmt.Println("Starting service...")
	handler.Handle()
}
