package main

import (
	"game/app"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	app := app.New()
	defer app.Close()
	app.Run()
}
