package camcontroller

import (
	"time"

	"github.com/WowVeryLogin/vulkan_engine/src/camcontroller/camera"
	"github.com/WowVeryLogin/vulkan_engine/src/window"
	"github.com/go-gl/glfw/v3.3/glfw"
	"gonum.org/v1/gonum/mat"
)

type Controller struct {
	lastUpdateTime  time.Time
	initialPosition mat.VecDense
}

func New(
	window *window.Window,
) *Controller {
	width, height := window.Window.GetSize()
	x := float64(width) / 2.0
	y := float64(height) / 2.0
	window.Window.SetCursorPos(x, y)
	x, y = window.Window.GetCursorPos()
	return &Controller{
		lastUpdateTime:  time.Now(),
		initialPosition: *mat.NewVecDense(2, []float64{x, y}),
	}
}

func (c *Controller) Update(
	window *window.Window,
	camera *camera.Camera,
) {
	since := time.Since(c.lastUpdateTime)
	c.lastUpdateTime = time.Now()

	var move [3]float32
	if window.Window.GetKey(glfw.KeyW) == glfw.Press {
		move[2] = 1.0
	}
	if window.Window.GetKey(glfw.KeyS) == glfw.Press {
		move[2] = -1.0
	}
	if window.Window.GetKey(glfw.KeyA) == glfw.Press {
		move[0] = -1.0
	}
	if window.Window.GetKey(glfw.KeyD) == glfw.Press {
		move[0] = 1.0
	}
	if window.Window.GetKey(glfw.KeyEscape) == glfw.Press {
		window.Window.SetShouldClose(true)
	}

	mouseChange := mat.NewVecDense(2, []float64{
		0, 0,
	})
	x, y := window.Window.GetCursorPos()
	newPosition := mat.NewVecDense(2, []float64{x, y})
	mouseChange.SubVec(&c.initialPosition, newPosition)
	pitch, yaw := -mouseChange.AtVec(1), mouseChange.AtVec(0)

	var moveVec mat.VecDense
	moveVec.ScaleVec(float64(since.Milliseconds())*0.001, mat.NewVecDense(3, []float64{
		float64(move[0]),
		float64(move[1]),
		float64(move[2]),
	}))
	camera.Move(moveVec.AtVec(0), moveVec.AtVec(1), moveVec.AtVec(2))
	camera.RotatePitch(pitch * float64(since.Milliseconds()) * 0.007)
	camera.RotateYaw(yaw * float64(since.Milliseconds()) * 0.007)

	window.Window.SetCursorPos(c.initialPosition.AtVec(0), c.initialPosition.AtVec(1))
}
