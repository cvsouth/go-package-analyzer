package main

import (
	"testing/data/simple_project/app"
	"testing/data/simple_project/util"
)

func main() {
	app.Run()
	message := util.Helper("CLI mode")
	println(message)
}
