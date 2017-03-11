package unicornify

import (
	"image"
	"image/color"
	"math"
)

func CircleShadingRGBA(x, y, r float64, col color.RGBA, shading float64) color.RGBA {
	if shading == 0 || y == 0{
		return col
	}
	sh1 := 1 - math.Sqrt(1 - y*y/(r*r))
	d := math.Sqrt(x*x+y*y)/r
	sh2 := math.Abs(y)/r// math.Sqrt(absy/r)
	sh := (1-d)*sh1 + d*sh2
	if y > 0 {
		return DarkenRGBA(col, uint8(255 * sh*shading))
	} else {
		return LightenRGBA(col, uint8(128 * sh*shading))
	}
}

func TopHalfCircleF(img *image.RGBA, cx, cy, r float64, col Color, shading float64) {
	circleImpl(img, int(cx+.5), int(cy+.5), int(r+.5), col, true, shading)
}

func CircleF(img *image.RGBA, cx, cy, r float64, col Color, shading float64) {
	Circle(img, int(cx+.5), int(cy+.5), int(r+.5), col, shading)
}

func Circle(img *image.RGBA, cx, cy, r int, col Color, shading float64) {
	circleImpl(img, cx, cy, r, col, false, shading)
}

func circleImpl(img *image.RGBA, cx, cy, r int, col Color, topHalfOnly bool, shading float64) {
	colrgba := color.RGBA{col.R, col.G, col.B, 255}
	imgsize := img.Bounds().Dx()
	if cx < -r || cy < -r || cx-r > imgsize || cy-r > imgsize {
		return
	}
	f := 1 - r
	ddF_x := 1
	ddF_y := -2 * r
	x := 0
	y := r

	fill := func(left, right, y int) {
		left += cx
		right += cx

		y += cy
		if left < 0 {
			left = 0
		}
		if right >= imgsize {
			right = imgsize - 1
		}

		for x := left; x <= right; x++ {
			thiscol := CircleShadingRGBA(float64(x - cx), float64(y - cy), float64(r), colrgba, shading)
			img.SetRGBA(x, y, thiscol)
		}
	}

	fill(-r, r, 0)

	for x < y {
		if f >= 0 {
			y--
			ddF_y += 2
			f += ddF_y
		}
		x++
		ddF_x += 2
		f += ddF_x
		fill(-x, x, -y)
		fill(-y, y, -x)
		if !topHalfOnly {
			fill(-x, x, y)
			fill(-y, y, x)
		}
	}
}

func between(v, min, max int) int {
	if min > max {
		min, max = max, min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round(v float64) int {
	return int(v + .5)
}
func ConnectCirclesF(img *image.RGBA, cx1, cy1, r1 float64, col1 Color, cx2, cy2, r2 float64, col2 Color, shading float64) {
	ConnectCircles(img, round(cx1), round(cy1), round(r1), col1, round(cx2), round(cy2), round(r2), col2, shading)
}
func ConnectCircles(img *image.RGBA, cx1, cy1, r1 int, col1 Color, cx2, cy2, r2 int, col2 Color, shading float64) {
	size := img.Bounds().Dx()
	xmin := between(cx1-r1, 0, cx2-r2)
	xmax := between(cx1+r1, cx2+r2, size)
	ymin := between(cy1-r1, 0, cy2-r2)
	ymax := between(cy1+r1, cy2+r2, size)
	cols := [256]color.RGBA{}
	for i := 0; i <= 255; i++ {
		mixed := MixColors(col1, col2, float64(i)/255)
		cols[i] = color.RGBA{mixed.R, mixed.G, mixed.B, 255}
	}
	d := r2 - r1
	vx := cx2 - cx1
	vy := cy2 - cy1
	a := float64(vx*vx + vy*vy - d*d)
	r1d := r1 * d
	r1s := r1 * r1

	d2xs := make([]int, xmax-xmin+1)
	for i := 0; i <= xmax-xmin; i++ {
		xcx2 := xmin + i - cx2
		d2xs[i] = xcx2 * xcx2
	}
	for y := ymin; y <= ymax; y++ {
		dy := y - cy1
		b_ := vy*dy + r1d
		c_ := dy*dy - r1s
		yc2y := y - cy2

		r2sdy2s := r2*r2 - yc2y*yc2y
		for x := xmin; x <= xmax; x++ {
			dx := x - cx1
			b := float64(-2 * (vx*dx + b_))
			c := float64(dx*dx + c_)
			var l float64
			if d2xs[x-xmin] <= r2sdy2s {
				l = 1
			} else if a == 0 {
				if b == 0 {
					continue
				}
				l = -c / b
			} else {
				p := b / a
				q := c / a
				disc := p*p/4 - q
				if disc < 0 {
					continue
				}
				sqrtdisc := math.Sqrt(disc)
				l = -p/2 + sqrtdisc
				if l > 1 {
					l -= 2 * sqrtdisc
				}
			}
			if l > 1 {
				dx2 := x - cx2
				dy2 := y - cy2
				if dx2*dx2+dy2*dy2 > r2*r2 {
					continue
				}
				l = 1
			}
			if l < 0 {
				if c > 0 {
					continue
				}
				l = 0
			}
			col := cols[int(l*255)]
			if (shading > 0) {
				tcy := float64(cy1) + l * float64(cy2-cy1)
				tcx := float64(cx1) + l * float64(cx2-cx1)
				tr := float64(r1) + l * float64(r2-r1)
				col = CircleShadingRGBA(float64(x) - tcx, float64(y) - tcy, tr, col, shading)
			}
			img.SetRGBA(x, y, col)
		}
	}
}
