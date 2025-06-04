package camera

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type Camera struct {
	fov, far, near float32
	matrix         *mat.Dense
	move           *mat.VecDense
	pitch, yaw     float64
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

func rotateVectorByQuaternion(v []float64, q []float64) []float64 {
	p := [4]float64{0, v[0], v[1], v[2]}

	// q * p * q_conjugate
	qp := quatMultiply(q[:], p[:])
	qConj := [4]float64{q[0], -q[1], -q[2], -q[3]}
	result := quatMultiply(qp, qConj[:])
	return result[1:]
}

func toColomnData(m *mat.Dense) [16]float32 {
	var result [16]float32
	for i := range 4 {
		for j, v := range m.RawRowView(i) {
			result[j*4+i] = float32(v)
		}
	}
	return result
}

func (c *Camera) ToMatrix() ([16]float32, [16]float32) {
	pitchQ := makeQuaternion(c.pitch, [3]float64{1, 0, 0})
	yawnQ := makeQuaternion(c.yaw, [3]float64{0, 1, 0})
	quat := quatMultiply(yawnQ[:], pitchQ[:])
	q0, q1, q2, q3 := quat[0], -quat[1], -quat[2], -quat[3]

	moveRotated := rotateVectorByQuaternion(c.move.RawVector().Data, []float64{q0, q1, q2, q3})

	view := mat.NewDense(4, 4, []float64{
		2.0*(q0*q0+q1*q1) - 1.0, 2.0 * (q1*q2 - q0*q3), 2.0 * (q1*q3 + q0*q2), moveRotated[0],
		2.0 * (q1*q2 + q0*q3), 2.0*(q0*q0+q2*q2) - 1.0, 2.0 * (q2*q3 - q0*q1), moveRotated[1],
		2.0 * (q1*q3 - q0*q2), 2.0 * (q2*q3 + q0*q1), 2.0*(q0*q0+q3*q3) - 1.0, moveRotated[2],
		0, 0, 0, 1,
	})

	return toColomnData(view), toColomnData(c.matrix)
}

func (c *Camera) Move(x, y, z float64) {
	moveLocal := []float64{-x, -y, -z}

	pitchQ := makeQuaternion(c.pitch, [3]float64{1, 0, 0})
	yawQ := makeQuaternion(c.yaw, [3]float64{0, 1, 0})
	quat := quatMultiply(yawQ[:], pitchQ[:])
	moveWorld := rotateVectorByQuaternion(moveLocal, quat)

	c.move.AddVec(c.move, mat.NewVecDense(3, moveWorld))
}

func quatMultiply(q1, q2 []float64) []float64 {
	a1, b1, c1, d1 := q1[0], q1[1], q1[2], q1[3]
	a2, b2, c2, d2 := q2[0], q2[1], q2[2], q2[3]

	return []float64{
		a1*a2 - b1*b2 - c1*c2 - d1*d2,
		a1*b2 + b1*a2 + c1*d2 - d1*c2,
		a1*c2 - b1*d2 + c1*a2 + d1*b2,
		a1*d2 + b1*c2 - c1*b2 + d1*a2,
	}
}

func (c *Camera) RotatePitch(angle float64) {
	c.pitch -= angle
	if c.pitch > 89 {
		c.pitch = 89
	}
	if c.pitch < -89 {
		c.pitch = -89
	}
}

func (c *Camera) RotateYaw(angle float64) {
	c.yaw -= angle
	c.yaw = math.Mod(c.yaw, 360.0)
	if c.yaw < 0 {
		c.yaw += 360.0
	}
}

func makeQuaternion(angle float64, vec [3]float64) [4]float64 {
	radians := angle * math.Pi / 180.0

	qvec := mat.NewVecDense(3, vec[:])
	norm := mat.Norm(qvec, 2) // Euclidean (L2) norm
	qvec.ScaleVec(1.0/norm*math.Sin(radians), qvec)

	// Multiply new quaternion onto current rotation
	return [4]float64{
		math.Cos(radians), qvec.AtVec(0), qvec.AtVec(1), qvec.AtVec(2),
	}
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
