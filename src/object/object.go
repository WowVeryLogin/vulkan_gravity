package object

import (
	"math"
	"time"

	"github.com/WowVeryLogin/vulkan_engine/src/object/model"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/pipeline"
	"gonum.org/v1/gonum/mat"
)

type GameObject struct {
	ID              int
	Model           *model.Model
	TextureType     int
	transformations mat.Dense
	offset          mat.VecDense
	onFrame         func(g *GameObject, since time.Duration)
	Mass            *model.MassModel
	Field           *model.FieldModel
}

type Position struct {
	X float64
	Y float64
	Z float64
}

type Transform interface {
	Transform(g *GameObject) *GameObject
}

var everIncreasingID int

func New(model *model.Model, textureType int) *GameObject {
	everIncreasingID += 1

	return &GameObject{
		ID:          everIncreasingID,
		Model:       model,
		TextureType: textureType,
		offset:      *mat.NewVecDense(3, []float64{0, 0, 0}),
		transformations: *mat.NewDense(3, 3, []float64{
			1, 0, 0,
			0, 1, 0,
			0, 0, 1,
		}),
	}
}

func (g *GameObject) GetPosition() [3]float32 {
	return [3]float32{
		float32(g.offset.AtVec(0)),
		float32(g.offset.AtVec(1)),
		float32(g.offset.AtVec(2)),
	}
}

func (g *GameObject) WithField(fieldModel model.FieldModel) *GameObject {
	g.Field = &fieldModel
	return g
}

func (g *GameObject) WithMass(massModel model.MassModel) *GameObject {
	g.Mass = &massModel
	return g
}

func (g *GameObject) WithInitialTranforms(initialTransforms []Transform) *GameObject {
	for _, transform := range initialTransforms {
		g = transform.Transform(g)
	}

	return g
}

func (g *GameObject) WithOnFrame(onFrame func(g *GameObject, since time.Duration)) *GameObject {
	g.onFrame = onFrame

	return g
}

func (g *GameObject) Rotate(degress float64, vec [3]float64) {
	*g = *NewRotate(degress, vec).Transform(g)
}

type Rotate struct {
	rotate mat.Dense
}

func NewRotate(degrees float64, vec [3]float64) Rotate {
	radians := degrees * math.Pi / 180.0

	qvec := mat.NewVecDense(3, vec[:])
	norm := mat.Norm(qvec, 2) // Euclidean (L2) norm
	qvec.ScaleVec(1.0/norm*math.Sin(radians), qvec)
	// Check for zero norm to avoid division by zero

	q0, q1, q2, q3 := math.Cos(radians), qvec.AtVec(0), qvec.AtVec(1), qvec.AtVec(2)

	return Rotate{
		rotate: *mat.NewDense(3, 3, []float64{
			2.0*(q0*q0+q1*q1) - 1.0, 2.0 * (q1*q2 - q0*q3), 2.0 * (q1*q3 + q0*q2),
			2.0 * (q1*q2 + q0*q3), 2.0*(q0*q0+q2*q2) - 1.0, 2.0 * (q2*q3 - q0*q1),
			2.0 * (q1*q3 - q0*q2), 2.0 * (q2*q3 + q0*q1), 2.0*(q0*q0+q3*q3) - 1.0,
		}),
	}
}

func (r Rotate) Transform(g *GameObject) *GameObject {
	result := mat.Dense{}
	result.Mul(&r.rotate, &g.transformations)
	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: result,
		offset:          g.offset,
		onFrame:         g.onFrame,
		TextureType:     g.TextureType,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) Scale(x float64, y float64, z float64) {
	*g = *NewScale(x, y, z).Transform(g)
}

type Scale struct {
	scale mat.Dense
}

func NewScale(x float64, y float64, z float64) Scale {
	return Scale{
		scale: *mat.NewDense(3, 3, []float64{
			x, 0, 0,
			0, y, 0,
			0, 0, z,
		}),
	}
}

func (s Scale) Transform(g *GameObject) *GameObject {
	result := mat.Dense{}
	result.Mul(&s.scale, &g.transformations)
	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: result,
		offset:          g.offset,
		onFrame:         g.onFrame,
		TextureType:     g.TextureType,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) Transition(x float64, y float64, z float64) {
	*g = *NewTransition(x, y, z).Transform(g)
}

type Transition struct {
	offset mat.VecDense
}

func NewTransition(x float64, y float64, z float64) Transition {
	return Transition{
		offset: *mat.NewVecDense(3, []float64{x, y, z}),
	}
}

func (t Transition) Transform(g *GameObject) *GameObject {
	result := mat.VecDense{}

	result.AddVec(&g.offset, &t.offset)

	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: g.transformations,
		offset:          result,
		onFrame:         g.onFrame,
		TextureType:     g.TextureType,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) ToPushData(since time.Duration) *pipeline.PushData {
	if g.onFrame != nil {
		g.onFrame(g, since)
	}

	transformation := [16]float32{}
	for i := range 3 {
		for j, v := range g.transformations.RawRowView(i) {
			transformation[j*4+i] = float32(v)
		}
		transformation[4*3+i] = float32(g.offset.AtVec(i))
	}
	transformation[15] = 1.0

	return &pipeline.PushData{
		Transformation: transformation,
		TextureType:    int32(g.TextureType),
	}
}
