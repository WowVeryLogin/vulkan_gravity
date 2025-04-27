package app

import (
	"game/device"
	"game/drawer"
	"game/gravity"
	"game/model"
	"game/object"
	"game/renderer"
	"game/window"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vulkan"
)

type App struct {
	window *window.Window
	device *device.Device

	models []*model.Model

	gameObjectsDrawer *drawer.Drawer
	gravity           *gravity.Gravity

	renderer    *renderer.Renderer
	gameObjects []*object.GameObject
}

func New() *App {
	window := window.New()

	vulkan.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vulkan.Init(); err != nil {
		panic("failed to initialize Vulkan: " + err.Error())
	}

	device := device.New(window)

	models, objects := loadGameObjects(device)

	gravity := gravity.New(device)
	gravity.UploadMassObjects(device, objects)
	gravity.UploadFieldObjects(device, objects)

	renderer := renderer.New(device, window.Extent)
	drawer := drawer.New(device, renderer.RenderPass, gravity.DescriptorsLayout)

	return &App{
		window:            &window,
		device:            device,
		gameObjectsDrawer: drawer,
		gravity:           gravity,
		renderer:          renderer,
		models:            models,
		gameObjects:       objects,
	}
}

func (a *App) Run() {
	for !a.window.ShouldClose() {
		glfw.PollEvents()

		if commandBuffer, frameIdx, err := a.renderer.BeginFrame(); err == nil {
			a.gravity.ComputeGravity(commandBuffer.ComputeCommandBuffer, frameIdx)
			computeFence, descriptors := a.gravity.ComputeGravityField(commandBuffer.ComputeCommandBuffer, frameIdx)
			a.renderer.BeginSwapChainRenderPass()
			a.gameObjectsDrawer.RenderGameObects(commandBuffer.GraphicsCommandBuffer, descriptors, a.gameObjects)
			a.renderer.EndSwapChainRenderPass()
			a.renderer.EndFrame(computeFence)

			if a.window.SizeChanged {
				for a.window.Extent.Height == 0 || a.window.Extent.Width == 0 {
					glfw.WaitEvents()
				}
				vulkan.DeviceWaitIdle(a.device.LogicalDevice)
				a.window.SizeChanged = false
				a.renderer.UpdateSwapchain(a.window.Extent)
			}
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
	a.gravity.Close()
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
	// triangle := model.New(device, []model.Vertex{
	// 	{Pos: model.Position{X: 0.0, Y: -0.5}, RGB: [3]float32{1, 0, 0}},
	// 	{Pos: model.Position{X: 0.5, Y: 0.5}, RGB: [3]float32{0, 1, 0}},
	// 	{Pos: model.Position{X: -0.5, Y: 0.5}, RGB: [3]float32{0, 0, 1}},
	// })

	rectangle := model.New(device, []model.Vertex{
		{Pos: model.Position{X: -0.5, Y: -0.5}, RGB: [3]float32{1, 0, 0}},
		{Pos: model.Position{X: 0.5, Y: 0.5}, RGB: [3]float32{0, 1, 0}},
		{Pos: model.Position{X: -0.5, Y: 0.5}, RGB: [3]float32{0, 1, 0}},
		{Pos: model.Position{X: -0.5, Y: -0.5}, RGB: [3]float32{0, 1, 0}},
		{Pos: model.Position{X: 0.5, Y: -0.5}, RGB: [3]float32{0, 0, 1}},
		{Pos: model.Position{X: 0.5, Y: 0.5}, RGB: [3]float32{1, 1, 0}},
	})

	circle := model.New(device, createCircleVertices(64))

	objects := []*object.GameObject{
		object.New(circle, [3]float32{1.0, 0.0, 0.0}).WithInitialTranforms([]object.Transform{
			object.NewScale(0.1, 0.1),
			object.NewTransition(0.5, 0.5),
		}).WithMass(model.MassModel{
			ID:       0,
			Mass:     1,
			Velocity: [2]float32{-0.5, 0.0},
		}),
		object.New(circle, [3]float32{0.0, 0.0, 1.0}).WithInitialTranforms([]object.Transform{
			object.NewScale(0.1, 0.1),
			object.NewTransition(-.45, -.25),
		}).WithMass(model.MassModel{
			ID:       1,
			Mass:     1,
			Velocity: [2]float32{0.5, 0.0},
		}),
	}

	for i := range 40 {
		for j := range 40 {
			objects = append(objects, object.New(rectangle, [3]float32{1.0, 1.0, 1.0}).WithInitialTranforms([]object.Transform{
				object.NewScale(0.005, 0.005),
				object.NewTransition(-1.0+(float64(i)+0.5)*2.0/40.0, -1.0+(float64(j)+0.5)*2.0/40.0),
			}).WithField(model.FieldModel{
				ID: i*40 + j,
			}))
		}
	}

	return []*model.Model{rectangle, circle}, objects
}

// func transformVertices(depth int, vertices []model.Vertex) []model.Vertex {
// 	upper := []model.Vertex{
// 		vertices[0],
// 		{
// 			Pos: model.Position{
// 				X: (vertices[0].Pos.X + vertices[1].Pos.X) * 0.5,
// 				Y: (vertices[0].Pos.Y + vertices[1].Pos.Y) * 0.5,
// 			},
// 			RGB: vertices[1].RGB,
// 		},
// 		{
// 			Pos: model.Position{
// 				X: (vertices[0].Pos.X + vertices[2].Pos.X) * 0.5,
// 				Y: (vertices[0].Pos.Y + vertices[2].Pos.Y) * 0.5,
// 			},
// 			RGB: vertices[2].RGB,
// 		},
// 	}

// 	right := []model.Vertex{
// 		{
// 			Pos: model.Position{
// 				X: (vertices[0].Pos.X + vertices[1].Pos.X) * 0.5,
// 				Y: (vertices[0].Pos.Y + vertices[1].Pos.Y) * 0.5,
// 			},
// 			RGB: vertices[0].RGB,
// 		},
// 		vertices[1],
// 		{
// 			Pos: model.Position{
// 				X: vertices[0].Pos.X,
// 				Y: vertices[1].Pos.Y,
// 			},
// 			RGB: vertices[2].RGB,
// 		},
// 	}

// 	left := []model.Vertex{
// 		{
// 			Pos: model.Position{
// 				X: (vertices[0].Pos.X + vertices[2].Pos.X) * 0.5,
// 				Y: (vertices[0].Pos.Y + vertices[2].Pos.Y) * 0.5,
// 			},
// 			RGB: vertices[0].RGB,
// 		},
// 		{
// 			Pos: model.Position{
// 				X: vertices[0].Pos.X,
// 				Y: vertices[1].Pos.Y,
// 			},
// 			RGB: vertices[1].RGB,
// 		},
// 		vertices[2],
// 	}

// 	newVertices := []model.Vertex{}
// 	if depth == 0 {
// 		newVertices = append(newVertices, upper...)
// 		newVertices = append(newVertices, right...)
// 		newVertices = append(newVertices, left...)
// 		return newVertices
// 	}

// 	newVertices = append(newVertices, transformVertices(depth-1, upper)...)
// 	newVertices = append(newVertices, transformVertices(depth-1, right)...)
// 	newVertices = append(newVertices, transformVertices(depth-1, left)...)
// 	return newVertices
// }
