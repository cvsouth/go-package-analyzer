package app

import (
	"fmt"
	"testing/data/simple_project/util"
)

// Run executes the application logic
func Run() {
	message := util.Helper("hello world")
	fmt.Println(message)
}
