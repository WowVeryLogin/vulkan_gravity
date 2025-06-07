package app

import (
	"math"
	"os"
	"time"

	"github.com/WowVeryLogin/vulkan_engine/src/object"
	"github.com/WowVeryLogin/vulkan_engine/src/object/model"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/descriptors"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/renderpass"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/texture"

	"github.com/WowVeryLogin/vulkan_engine/src/camcontroller"
	"github.com/WowVeryLogin/vulkan_engine/src/camcontroller/camera"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/buffer"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/swapchain"
	"github.com/WowVeryLogin/vulkan_engine/src/systems/renderer"
	pointlights "github.com/WowVeryLogin/vulkan_engine/src/systems/renderer/point_lights"
	"github.com/WowVeryLogin/vulkan_engine/src/window"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vulkan"
)

type App struct {
	window *window.Window
	device *device.Device

	models []*model.Model

	gameObjectsRenderer *renderer.Renderer
	pointLightsRenderer *pointlights.PointLightsRenderer

	renderPass  *renderpass.RenderPass
	gameObjects []*object.GameObject
	camera      *camera.Camera

	cameraController *camcontroller.Controller

	descriptorManager   descriptors.SetsManager
	perFrameDescriptors []*descriptors.DescriptorSet
	cameraUbos          []*buffer.Buffer[camera.CameraUbo]
}

func New() *App {
	window := window.New()

	vulkan.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vulkan.Init(); err != nil {
		panic("failed to initialize Vulkan: " + err.Error())
	}

	device := device.New(window)
	models, objects := loadGameObjects(device)

	renderpass := renderpass.New(device, window.Extent)
	cam := camera.New(50.0, renderpass.GetAspectRatio(), 0.1, 10.0)

	viewUbo := make([]*buffer.Buffer[camera.CameraUbo], swapchain.MAX_FRAMES_IN_FLIGHT)
	sets := []*descriptors.DescriptorSet{}
	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		viewUbo[i] = buffer.New[camera.CameraUbo](
			device, 1, true,
			vulkan.BufferUsageFlags(vulkan.BufferUsageUniformBufferBit),
			vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit),
		)

		descrptrs := []descriptors.Descriptor{
			{
				Type:   vulkan.DescriptorTypeUniformBuffer,
				Flags:  vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit | vulkan.ShaderStageFragmentBit),
				Buffer: viewUbo[i].Buffer,
			},
		}

		for _, m := range models {
			descrptrs = append(descrptrs, descriptors.Descriptor{
				Type:         vulkan.DescriptorTypeCombinedImageSampler,
				Flags:        vulkan.ShaderStageFlags(vulkan.ShaderStageFragmentBit),
				ImageView:    m.Texture.ImageView,
				ImageSampler: m.Texture.Sampler,
			})
		}

		sets = append(sets, &descriptors.DescriptorSet{
			Descriptors: descrptrs,
		})
	}
	descriptorsMngr := descriptors.NewSets(device, sets)

	layouts := []vulkan.DescriptorSetLayout{}
	for _, set := range sets {
		layouts = append(layouts, set.Layout)
	}
	objectRenderer := renderer.New(device, renderpass.RenderPass, layouts)

	pointLightRenderer := pointlights.New(device, renderpass.RenderPass, layouts)

	return &App{
		window: window,
		device: device,

		pointLightsRenderer: pointLightRenderer,
		gameObjectsRenderer: objectRenderer,

		renderPass:       renderpass,
		camera:           cam,
		cameraController: camcontroller.New(window),
		models:           models,
		gameObjects:      objects,

		perFrameDescriptors: sets,
		cameraUbos:          viewUbo,
		descriptorManager:   descriptorsMngr,
	}
}

func (a *App) Run() {
	for !a.window.ShouldClose() {
		glfw.PollEvents()

		a.cameraController.Update(a.window, a.camera)

		commandBuffer, err := a.renderPass.BeginFrame()
		if err == nil {
			frameIdx := a.renderPass.FrameIdx
			a.cameraUbos[frameIdx].WriteBuffer([]camera.CameraUbo{
				a.camera.Ubo(),
			})
			a.renderPass.BeginSwapChainRenderPass()

			a.gameObjectsRenderer.Render(
				commandBuffer.GraphicsCommandBuffer,
				a.gameObjects,
				a.perFrameDescriptors[frameIdx],
			)
			a.pointLightsRenderer.Render(
				commandBuffer.GraphicsCommandBuffer,
				a.perFrameDescriptors[frameIdx],
			)

			a.renderPass.EndSwapChainRenderPass()
			err = a.renderPass.EndFrame()
		}

		if err == swapchain.ErrOutOfDate || a.window.SizeChanged {
			for a.window.Extent.Height == 0 || a.window.Extent.Width == 0 {
				glfw.WaitEvents()
			}
			vulkan.DeviceWaitIdle(a.device.LogicalDevice)
			a.window.SizeChanged = false
			a.renderPass.UpdateSwapchain(a.window.Extent)
			a.camera.Update(a.renderPass.GetAspectRatio())
		}
	}

	if err := vulkan.Error(vulkan.DeviceWaitIdle(a.device.LogicalDevice)); err != nil {
		panic("failed to wait for finish: " + err.Error())
	}
}

func (a *App) Close() {
	for _, desc := range a.perFrameDescriptors {
		desc.Close()
	}
	for _, ubo := range a.cameraUbos {
		ubo.Close()
	}
	a.descriptorManager.Close()

	for _, model := range a.models {
		model.Close()
	}
	a.gameObjectsRenderer.Close()
	a.renderPass.Close()
	a.pointLightsRenderer.Close()
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
	// cube := model.New(device, []model.Vertex{
	// 	//left
	// 	{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
	// 	{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},
	// 	{Pos: model.Position{-.5, -.5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},
	// 	{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
	// 	{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{0.9, 0.9, 0.9}},
	// 	{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{0.9, 0.9, 0.9}},

	// 	//right
	// 	{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.8, .8, .1}},
	// 	{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .8, .1}},
	// 	{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.8, .8, .1}},
	// 	{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.8, .8, .1}},
	// 	{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.8, .8, .1}},
	// 	{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .8, .1}},

	// 	// top
	// 	{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
	// 	{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},
	// 	{Pos: model.Position{-.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},
	// 	{Pos: model.Position{-.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
	// 	{Pos: model.Position{.5, -.5, -.5}, RGB: [3]float32{.9, .6, .1}},
	// 	{Pos: model.Position{.5, -.5, .5}, RGB: [3]float32{.9, .6, .1}},

	// 	// bottom
	// 	{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
	// 	{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},
	// 	{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},
	// 	{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
	// 	{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.8, .1, .1}},
	// 	{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.8, .1, .1}},

	// 	//nose
	// 	{Pos: model.Position{-.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
	// 	{Pos: model.Position{.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},
	// 	{Pos: model.Position{-.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},
	// 	{Pos: model.Position{-.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
	// 	{Pos: model.Position{.5, -.5, 0.5}, RGB: [3]float32{.1, .1, .8}},
	// 	{Pos: model.Position{.5, .5, 0.5}, RGB: [3]float32{.1, .1, .8}},

	// 	// tail
	// 	{Pos: model.Position{-.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// 	{Pos: model.Position{.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// 	{Pos: model.Position{-.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// 	{Pos: model.Position{-.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// 	{Pos: model.Position{.5, -.5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// 	{Pos: model.Position{.5, .5, -0.5}, RGB: [3]float32{.1, .8, .1}},
	// })

	avocado := model.NewWithGLTF(device, "assets/avocado.glb")

	file, err := os.Open("assets/image.png")
	if err != nil {
		panic("failed to open floor file: " + err.Error())
	}
	defer file.Close()
	floorTexture := texture.TextureConfigFromPNG(file)

	floor := model.New(device, []model.Vertex{
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{0, 0}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{1, 1}},
		{Pos: model.Position{-.5, .5, .5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{1, 0}},
		{Pos: model.Position{-.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{0, 0}},
		{Pos: model.Position{.5, .5, -.5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{0, 1}},
		{Pos: model.Position{.5, .5, .5}, RGB: [3]float32{.72, .72, .72}, Normal: [3]float32{0, -1, 0}, UV: [2]float32{1, 1}},
	}, []uint32{
		0, 1, 2,
		3, 4, 5,
	}, floorTexture)

	objects := []*object.GameObject{
		// object.New(cube, [3]float32{0.0, 0.0, 0.0}).WithInitialTranforms([]object.Transform{
		// 	object.NewScale(0.3, 0.3, 0.3),
		// 	object.NewTransition(0.0, 0.0, 2.5),
		// }).WithOnFrame(func(g *object.GameObject, since time.Duration) {
		// 	g.Rotate(0.5*float64(since.Milliseconds())/15.0, [3]float64{1, 1, 1})
		// }),
		object.New(avocado, 0).WithInitialTranforms([]object.Transform{
			object.NewScale(10.0, 10.0, 10.0),
			object.NewTransition(0.0, -1.0, 2.5),
		}).WithOnFrame(func(g *object.GameObject, since time.Duration) {
			g.Rotate(0.5*float64(since.Milliseconds())/15.0, [3]float64{0, 1, 0})
		}),
		object.New(floor, 1).WithInitialTranforms([]object.Transform{
			object.NewScale(5.0, 5.0, 5.0),
			object.NewTransition(0.0, -2.0, 2.5),
		}),
	}

	return []*model.Model{avocado, floor}, objects
}
