package window

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vulkan"
)

type Window struct {
	window      *glfw.Window
	Extent      vulkan.Extent2D
	SizeChanged bool
}

func New() Window {
	if err := glfw.Init(); err != nil {
		panic("failed to initialize GLFW: " + err.Error())
	}
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	window, err := glfw.CreateWindow(800, 600, "Game App", nil, nil)
	if err != nil {
		panic(err)
	}

	w := Window{
		window: window,
		Extent: vulkan.Extent2D{
			Width:  800,
			Height: 600,
		},
	}

	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width int, height int) {
		w.Extent.Height = uint32(height)
		w.Extent.Width = uint32(width)
		w.SizeChanged = true
	})

	return w
}

func (w *Window) Close() {
	w.window.Destroy()
	glfw.Terminate()
}

func (w *Window) ShouldClose() bool {
	return w.window.ShouldClose()
}

func (w *Window) CreateSurface(instance vulkan.Instance) vulkan.Surface {
	surface, err := w.window.CreateWindowSurface(instance, nil)
	if err != nil {
		panic("failed to create window surface: " + err.Error())
	}

	return vulkan.SurfaceFromPointer(surface)
}

func (w *Window) GetRequiredInstanceExtensions() []string {
	var result []string
	for _, e := range w.window.GetRequiredInstanceExtensions() {
		result = append(result, e+"\x00")
	}

	return result
}
