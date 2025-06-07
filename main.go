package main

import (
	"runtime"

	"github.com/WowVeryLogin/vulkan_engine/src/app"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	app := app.New()
	defer app.Close()
	app.Run()
}
