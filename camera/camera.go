package camera

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type Camera struct {
	fov, far, near float32
	matrix         *mat.Dense
	move           *mat.VecDense
	rotate         *mat.Dense
}

func New(fov float32, aspect float32, near float32, far float32) *Camera {
	c := &Camera{
		fov:  fov,
		far:  far,
		near: near,
		move: mat.NewVecDense(3, []float64{0, 0, 0}),
	}
	c.Update(aspect)
	return c
}

func (c *Camera) ToMatrix() [16]float32 {
	var result mat.Dense
	result.Mul(c.matrix, mat.NewDense(4, 4, []float64{
		1, 0, 0, c.move.AtVec(0),
		0, 1, 0, c.move.AtVec(1),
		0, 0, 1, c.move.AtVec(2),
		0, 0, 0, 1,
	}))

	var camera [16]float32
	for i := range 4 {
		for j, v := range result.RawRowView(i) {
			camera[j*4+i] = float32(v)
		}
	}
	return camera
}

func (c *Camera) Move(x, y, z float64) {
	c.move.AddVec(c.move, mat.NewVecDense(3, []float64{
		float64(-x), float64(-y), float64(-z),
	}))
}

func (c *Camera) Rotate(angle [3]float32) {
}

func (c *Camera) Update(aspect float32) {
	s := 1.0 / math.Tan(float64(c.fov)*0.5*math.Pi/180.0)
	c.matrix = mat.NewDense(4, 4, []float64{
		s / float64(aspect), 0, 0, 0,
		0, s, 0, 0,
		0, 0, float64(c.far) / float64(c.far-c.near), float64(c.far*c.near) / float64(c.near-c.far),
		0, 0, 1, 0,
	})
}
