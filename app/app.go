package app

import (
	"game/camcontroller"
	"game/camera"
	"game/device"
	"game/drawer"
	"game/model"
	"game/object"
	"game/renderer"
	"game/swapchain"
	"game/window"
	"math"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vulkan"
)

type App struct {
	window *window.Window
	device *device.Device

	models []*model.Model

	gameObjectsDrawer *drawer.Drawer

	renderer    *renderer.Renderer
	gameObjects []*object.GameObject
	camera      *camera.Camera

	cameraController *camcontroller.Controller
}

func New() *App {
	window := window.New()

	vulkan.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vulkan.Init(); err != nil {
		panic("failed to initialize Vulkan: " + err.Error())
	}

	device := device.New(window)

	models, objects := loadGameObjects(device)

	renderer := renderer.New(device, window.Extent)
	drawer := drawer.New(device, renderer.RenderPass)
	camera := camera.New(50.0, renderer.GetAspectRatio(), 0.1, 10.0)

	return &App{
		window:            window,
		device:            device,
		gameObjectsDrawer: drawer,
		renderer:          renderer,
		camera:            camera,
		cameraController:  camcontroller.New(window),
		models:            models,
		gameObjects:       objects,
	}
}

func (a *App) Run() {
	for !a.window.ShouldClose() {
		glfw.PollEvents()

		a.cameraController.Update(a.window, a.camera)

		commandBuffer, _, err := a.renderer.BeginFrame()
		if err == nil {
			a.renderer.BeginSwapChainRenderPass()
			a.gameObjectsDrawer.RenderGameObects(commandBuffer.GraphicsCommandBuffer, a.gameObjects, a.camera)
			a.renderer.EndSwapChainRenderPass()
			err = a.renderer.EndFrame()
		}

		if err == swapchain.ErrOutOfDate || a.window.SizeChanged {
			for a.window.Extent.Height == 0 || a.window.Extent.Width == 0 {
				glfw.WaitEvents()
			}
			vulkan.DeviceWaitIdle(a.device.LogicalDevice)
			a.window.SizeChanged = false
			a.renderer.UpdateSwapchain(a.window.Extent)
			a.camera.Update(a.renderer.GetAspectRatio())
		}
	}

	if err := vulkan.Error(vulkan.DeviceWaitIdle(a.device.LogicalDevice)); err != nil {
		panic("failed to wait for finish: " + err.Error())
	}
}

func (a *App) Close() {
	for _, model := range a.models {
		model.Close()
	}
	a.gameObjectsDrawer.Close()
	a.renderer.Close()
	a.device.Close()
	a.window.Close()
}

func createCircleVertices(numSides int) []model.Vertex {
	vertices := make([]model.Vertex, numSides+1)
	angleStep := 2 * math.Pi / float64(numSides)

	for i := 0; i < numSides; i++ {
		angle := float64(i) * angleStep
		x := 0.5 * float32(math.Cos(angle))
		y := 0.5 * float32(math.Sin(angle))
		vertices[i] = model.Vertex{
			Pos: model.Position{X: x, Y: y},
			RGB: [3]float32{1, 1, 1},
		}
	}

	vertices[numSides] = model.Vertex{
		Pos: model.Position{X: 0, Y: 0},
		RGB: [3]float32{1, 1, 1},
	}

	var result []model.Vertex
	for i := range vertices {
		result = append(result, vertices[i], vertices[(i+1)%numSides], vertices[numSides])
	}
	return result
}

func loadGameObjects(device *device.Device) ([]*model.Model, []*object.GameObject) {
	cube := model.New(device, []model.Vertex{
		//left
		{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
		{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},
		{Pos: model.Position{-.5, -.5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},
		{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
		{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},

		//right
		{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.8, .8, .1}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .8, .1}},
		{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.8, .8, .1}},
		{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.8, .8, .1}},
		{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.8, .8, .1}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .8, .1}},

		// top
		{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
		{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},
		{Pos: model.Position{-.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},
		{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
		{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
		{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},

		// bottom
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},
		{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
		{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},

		//nose
		{Pos: model.Position{-.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
		{Pos: model.Position{.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},
		{Pos: model.Position{-.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},
		{Pos: model.Position{-.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
		{Pos: model.Position{.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
		{Pos: model.Position{.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},

		// tail
		{Pos: model.Position{-.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
		{Pos: model.Position{.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
		{Pos: model.Position{-.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
		{Pos: model.Position{-.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
		{Pos: model.Position{.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
		{Pos: model.Position{.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	})

	floor := model.New(device, []model.Vertex{
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.72, .72, .72}},
		{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{.72, .72, .72}},
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}},
		{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.72, .72, .72}},
	})

	objects := []*object.GameObject{
		object.New(cube, [3]float32{0.0, 0.0, 0.0}).WithInitialTranforms([]object.Transform{
			object.NewScale(0.3, 0.3, 0.3),
			object.NewTransition(0.0, 0.0, 2.5),
		}).WithOnFrame(func(g *object.GameObject, since time.Duration) {
			g.Rotate(0.5*float64(since.Milliseconds())/15.0, [3]float64{1, 1, 1})
		}),
		object.New(floor, [3]float32{0.0, 0.0, 0.0}).WithInitialTranforms([]object.Transform{
			object.NewScale(5.0, 5.0, 5.0),
			object.NewTransition(0.0, -2.0, 2.5),
		}),
	}

	return []*model.Model{cube, floor}, objects
}
