package camcontroller

import (
	"game/camera"
	"game/window"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"gonum.org/v1/gonum/mat"
)

type Controller struct {
	lastUpdateTime time.Time
}

func New() *Controller {
	return &Controller{
		lastUpdateTime: time.Now(),
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

	var moveVec mat.VecDense
	moveVec.ScaleVec(float64(since.Milliseconds())*0.001, mat.NewVecDense(3, []float64{
		float64(move[0]),
		float64(move[1]),
		float64(move[2]),
	}))
	camera.Move(moveVec.AtVec(0), moveVec.AtVec(1), moveVec.AtVec(2))
}
