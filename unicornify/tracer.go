package unicornify

import (
	"image"
	"image/color"
	"math"
)

type Tracer interface {
	Trace(x, y int) (bool, float64, Point3d, Color)
	GetBounds() image.Rectangle
}

type WrappingTracer interface {
	Tracer
	Add(tracers ...Tracer)
}

func DrawTracer(t Tracer, img *image.RGBA, yCallback func(int)) {
	r := img.Bounds().Intersect(t.GetBounds())
	for y := r.Min.Y; y <= r.Max.Y; y++ {
		for x := r.Min.X; x <= r.Max.X; x++ {
			any, _, _, col := t.Trace(x, y)
			if any {
				img.Set(x, y, col)
			}
		}
		if yCallback != nil {
			yCallback(y)
		}
	}
}

// ------- GroupTracer -------

type GroupTracer struct {
	tracers       []Tracer
	bounds        image.Rectangle
	boundsCurrent bool
}

func NewGroupTracer() *GroupTracer {
	return &GroupTracer{}
}

func (gt *GroupTracer) Trace(x, y int) (bool, float64, Point3d, Color) {
	any := false
	var minz float64 = 0.0
	var col Color = Black
	var dir Point3d
	for _, t := range gt.tracers {
		tr := t.GetBounds()
		if x < tr.Min.X || x > tr.Max.X || y < tr.Min.Y || y > tr.Max.Y {
			continue
		}
		ok, z, thisdir, thiscol := t.Trace(x, y)
		if ok {
			if !any || z < minz {
				col = thiscol
				minz = z
				dir = thisdir
				any = true
			}
		}
	}
	return any, minz, dir, col
}

func (gt *GroupTracer) GetBounds() image.Rectangle {
	if !gt.boundsCurrent {
		if len(gt.tracers) == 0 {
			gt.bounds = image.Rect(-10, -10, -10, -10)
		} else {
			r := gt.tracers[0].GetBounds()
			for _, t := range gt.tracers[1:] {
				r = r.Union(t.GetBounds())
			}
			gt.bounds = r
		}
		gt.boundsCurrent = true
	}
	return gt.bounds
}

func (gt *GroupTracer) Add(ts ...Tracer) {
	for _, t := range ts {
		gt.tracers = append(gt.tracers, t)
	}
	gt.boundsCurrent = false
}

// ------- ImageTracer -------

type ImageTracer struct {
	img    *image.RGBA
	bounds image.Rectangle
	z      float64
}

var NoDirection = Point3d{0, 0, 0}

func (t *ImageTracer) Trace(x, y int) (bool, float64, Point3d, Color) {
	tr := t.bounds
	if x < tr.Min.X || x > tr.Max.X || y < tr.Min.Y || y > tr.Max.Y {
		return false, 0, NoDirection, Black
	}
	c := t.img.At(x, y).(color.RGBA)
	if c.A < 255 {
		return false, 0, NoDirection, Black
	}
	return true, t.z, NoDirection, Color{c.R, c.G, c.B}
}

func (t *ImageTracer) GetBounds() image.Rectangle {
	return t.bounds
}

// ------- DirectionalLightTracer -------

type DirectionalLightTracer struct {
	GroupTracer
	LightDirectionUnit Point3d
}

func (t *DirectionalLightTracer) Trace(x, y int) (bool, float64, Point3d, Color) {
	ok, z, dir, col := t.GroupTracer.Trace(x, y)
	if !ok {
		return ok, z, dir, col
	}
	dirlen := dir.Length()
	if dirlen == 0 {
		return ok, z, dir, col
	}

	unit := dir.Times(1 / dirlen)
	sp := unit.ScalarProd(t.LightDirectionUnit)

	if sp >= 0 {
		col = Darken(col, uint8(sp*96))
	} else {
		col = Lighten(col, uint8(-sp*48))
	}

	return ok, z, dir, col
}

func (t *DirectionalLightTracer) Add(ts ...Tracer) {
	t.GroupTracer.Add(ts...)
}
func (t *DirectionalLightTracer) SetLightDirection(dir Point3d) {
	length := dir.Length()
	if length != 0 {
		dir = dir.Times(1 / length)
	}
	t.LightDirectionUnit = dir
}

func NewDirectionalLightTracer(lightDirection Point3d) *DirectionalLightTracer {
	result := &DirectionalLightTracer{}
	result.SetLightDirection(lightDirection)
	return result
}

// ------- PointLightTracer (experimental, unused) -------

type PointLightTracer struct {
	LightPositions []Point3d
	SourceTracer   Tracer
	HalfLifes      []float64
}

func (t *PointLightTracer) Trace(x, y int) (bool, float64, Point3d, Color) {
	ok, z, dir, col := t.SourceTracer.Trace(x, y)
	if !ok {
		return ok, z, dir, col
	}
	dirlen := dir.Length()
	unit := Point3d{0, 0, 0}
	if dirlen > 0 {
		unit = dir.Times(1 / dirlen)
	} else {
		return ok, z, dir, col
	}

	lightsum := 0.0
	for i, lightposition := range t.LightPositions {
		lightdir := Point3d{float64(x), float64(y), z}.Shifted(lightposition.Neg())
		lightdirunit := lightdir.Times(1 / lightdir.Length())

		sp := -unit.ScalarProd(lightdirunit)
		if dirlen == 0 {
			sp = 0.5
		}
		if sp < 0 {
			continue
		}
		strength := math.Pow(0.5, lightdir.Length()/t.HalfLifes[i])
		sp = sp * strength
		lightsum += sp
	}

	if lightsum < 0 {
		col = Black
	} else {
		if lightsum <= 0.5 {
			col = Darken(col, uint8((0.5-lightsum)*2*255))
		} else {
			col = Lighten(col, uint8((lightsum-0.5)*96))
		}
	}

	return ok, z, dir, col
}

func (t *PointLightTracer) GetBounds() image.Rectangle {
	return t.SourceTracer.GetBounds()
}

func NewPointLightTracer(source Tracer, lightPos ...Point3d) *PointLightTracer {
	result := &PointLightTracer{SourceTracer: source, LightPositions: lightPos}
	return result
}
