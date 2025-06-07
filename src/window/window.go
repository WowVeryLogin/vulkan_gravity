package window

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vulkan"
)

type Window struct {
	Window      *glfw.Window
	Extent      vulkan.Extent2D
	SizeChanged bool
}

func New() *Window {
	if err := glfw.Init(); err != nil {
		panic("failed to initialize GLFW: " + err.Error())
	}
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	// monitor := glfw.GetPrimaryMonitor()

	window, err := glfw.CreateWindow(800, 600, "Game App", nil, nil)
	if err != nil {
		panic(err)
	}
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	w := &Window{
		Window: window,
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
	w.Window.Destroy()
	glfw.Terminate()
}

func (w *Window) ShouldClose() bool {
	return w.Window.ShouldClose()
}

func (w *Window) CreateSurface(instance vulkan.Instance) vulkan.Surface {
	surface, err := w.Window.CreateWindowSurface(instance, nil)
	if err != nil {
		panic("failed to create window surface: " + err.Error())
	}

	return vulkan.SurfaceFromPointer(surface)
}

func (w *Window) GetRequiredInstanceExtensions() []string {
	var result []string
	for _, e := range w.Window.GetRequiredInstanceExtensions() {
		result = append(result, e+"\x00")
	}

	return result
}
