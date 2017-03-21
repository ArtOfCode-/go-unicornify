package unicornify

import (
	"image"
)

type Thing interface {
	Project(wv WorldView)
	Bounding() image.Rectangle
	GetTracer(img *image.RGBA, wv WorldView) Tracer
}

type Figure struct {
	things []Thing
}

func (f *Figure) Add(things ...Thing) {
	f.things = append(f.things, things...)
}

func (f *Figure) Project(wv WorldView) {
	for i := 0; i < len(f.things); i++ {
		f.things[i].Project(wv)
	}
}

type FigureTracer struct {
	*GroupTracer
	f   *Figure
	r   image.Rectangle
	img *image.RGBA
	wv  WorldView
}

func (f *Figure) GetTracer(img *image.RGBA, wv WorldView) Tracer {
	gt := NewGroupTracer()

	for _, th := range f.things {
		gt.Add(th.GetTracer(img, wv))
	}
	return &FigureTracer{GroupTracer: gt, f: f, img: img, wv: wv}
}

func (f *Figure) Draw(img *image.RGBA, wv WorldView, wrappingTracer WrappingTracer, yCallback func(int), additionalTracers ...Tracer) {
	tracer := f.GetTracer(img, wv).(*FigureTracer)
	tracer.Add(additionalTracers...)

	var final Tracer = tracer

	if wrappingTracer != nil {
		wrappingTracer.Add(tracer)
		final = wrappingTracer
	}

	DrawTracer(final, img, yCallback)
}

func (f *Figure) Bounding() image.Rectangle {
	result := image.Rect(0, 0, 0, 0)
	for _, t := range f.things {
		result = result.Union(t.Bounding())
	}
	return result
}

func (f *Figure) Scale(factor float64) {
	for b := range f.BallSet() {
		b.Radius *= factor
		b.Center = b.Center.Times(factor)
	}
}

func (f *Figure) BallSet() <-chan *Ball {
	seen := make(map[*Ball]bool)
	ch := make(chan *Ball)
	go ballSetImpl(f, seen, ch, true)
	return ch
}

func ballSetImpl(t Thing, seen map[*Ball]bool, ch chan *Ball, outer bool) {
	switch t := t.(type) {
	case *Ball:
		if !seen[t] {
			ch <- t
			seen[t] = true
		}
	case *Bone:
		ballSetImpl(t.Balls[0], seen, ch, false)
		ballSetImpl(t.Balls[1], seen, ch, false)
	case *Figure:
		for _, s := range t.things {
			ballSetImpl(s, seen, ch, false)
		}
	/*case *Quad:
	for _, b := range t.Balls {
		ballSetImpl(b, seen, ch, false)
	}*/
	default:
		panic("unhandled thing type")
	}
	if outer {
		close(ch)
	}
}
