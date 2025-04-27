package object

import (
	"game/model"
	"game/pipeline"
	"math"
	"time"

	"gonum.org/v1/gonum/mat"
)

type GameObject struct {
	ID              int
	Model           *model.Model
	color           [3]float32
	transformations mat.Dense
	offset          mat.Vector
	onFrame         func(g *GameObject, since time.Duration)
	Mass            *model.MassModel
	Field           *model.FieldModel
}

type Position struct {
	X float64
	Y float64
}

type Transform interface {
	Transform(g *GameObject) *GameObject
}

var everIncreasingID int

func New(model *model.Model, color [3]float32) *GameObject {
	everIncreasingID += 1

	return &GameObject{
		ID:              everIncreasingID,
		Model:           model,
		color:           color,
		transformations: *mat.NewDense(2, 2, []float64{1, 0, 0, 1}),
		offset:          mat.NewVecDense(2, []float64{0, 0}),
	}
}

func (g *GameObject) GetPosition() [2]float32 {
	return [2]float32{
		float32(g.offset.AtVec(0)),
		float32(g.offset.AtVec(1)),
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

func (g *GameObject) Position() Position {
	return Position{
		X: g.offset.AtVec(0),
		Y: g.offset.AtVec(1),
	}
}

func (g *GameObject) Rotate(degress float64) {
	*g = *NewRotate(degress).Transform(g)
}

type Rotate struct {
	rotate mat.Dense
}

func NewRotate(degrees float64) Rotate {
	s := math.Sin(degrees * math.Pi / 180)
	c := math.Cos(degrees * math.Pi / 180)
	return Rotate{
		rotate: *mat.NewDense(2, 2, []float64{
			c, -s, s, c,
		}),
	}
}

func (r Rotate) Transform(g *GameObject) *GameObject {
	result := mat.Dense{}
	result.Mul(&g.transformations, &r.rotate)
	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: result,
		offset:          g.offset,
		onFrame:         g.onFrame,
		color:           g.color,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) Scale(x float64, y float64) {
	g = NewScale(x, y).Transform(g)
}

type Scale struct {
	scale mat.Dense
}

func NewScale(x float64, y float64) Scale {
	return Scale{
		scale: *mat.NewDense(2, 2, []float64{
			x, 0, 0, y,
		}),
	}
}

func (s Scale) Transform(g *GameObject) *GameObject {
	result := mat.Dense{}
	result.Mul(&g.transformations, &s.scale)
	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: result,
		offset:          g.offset,
		onFrame:         g.onFrame,
		color:           g.color,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) Transition(x float64, y float64) {
	*g = *NewTransition(x, y).Transform(g)
}

type Transition struct {
	offset mat.VecDense
}

func NewTransition(x float64, y float64) Transition {
	return Transition{
		offset: *mat.NewVecDense(2, []float64{x, y}),
	}
}

func (t Transition) Transform(g *GameObject) *GameObject {
	result := mat.VecDense{}
	result.AddVec(g.offset, &t.offset)
	return &GameObject{
		ID:              g.ID,
		Model:           g.Model,
		transformations: g.transformations,
		offset:          &result,
		onFrame:         g.onFrame,
		color:           g.color,
		Mass:            g.Mass,
		Field:           g.Field,
	}
}

func (g *GameObject) ToPushData(since time.Duration) *pipeline.PushData {
	if g.onFrame != nil {
		g.onFrame(g, since)
	}

	transformation := [4]float32{}
	for i, v := range g.transformations.RawMatrix().Data {
		transformation[i] = float32(v)
	}

	offset := [2]float32{}
	offset[0] = float32(g.offset.AtVec(0))
	offset[1] = float32(g.offset.AtVec(1))

	isField := uint32(0)
	id := g.ID
	if g.Field != nil {
		isField = 1
		id = g.Field.ID
	} else {
		isField = 0
		id = g.Mass.ID
	}

	return &pipeline.PushData{
		Transformation: transformation,
		Offset:         offset,
		Color:          g.color,
		IsField:        isField,
		Index:          uint32(id),
	}
}
