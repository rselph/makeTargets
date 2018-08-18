package main

import (
	"image/color"
	"math"
	"image"
	"sync"
)

// See http://blog.ivank.net/fastest-gaussian-blur.html
func gaussianBlur(in image.Image, radius float64) image.Image {
	bxs := boxesForGauss(radius, 3)

	out := image.NewRGBA64(in.Bounds())
	tmp := image.NewRGBA64(in.Bounds())
	scratch := image.NewRGBA64(in.Bounds())

	boxBlur(in, scratch, out, (bxs[0]-1)/2)
	boxBlur(out, scratch, tmp, (bxs[1]-1)/2)
	boxBlur(tmp, scratch, out, (bxs[2]-1)/2)

	return out
}

func boxBlur(in image.Image, scratch, out *image.RGBA64, r int) {
	boxBlurHorizontal(in, scratch, r)
	boxBlurVertical(scratch, out, r)
}

func boxBlurHorizontal(in image.Image, out *image.RGBA64, r int) {
	//var iarr = 1 / (r+r+1);
	var (
		iarr   = 1.0 / float64(2*r+1)
		top    = in.Bounds().Min.Y
		bottom = in.Bounds().Max.Y
		left   = in.Bounds().Min.X
		right  = in.Bounds().Max.X
	)

	wg := sync.WaitGroup{}
	//for(var i=0; i<h; i++) {
	for y := top; y < bottom; y++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			//var ti = i*w, li = ti, ri = ti+r;
			//var fv = scl[ti], lv = scl[ti+w-1], val = (r+1)*fv;
			var (
				tx  = left
				lx  = left
				rx  = left + r
				fv  = newColorVal(in.At(left, y))
				lv  = newColorVal(in.At(right-1, y))
				val = fv.times(float64(r + 1))
			)

			//for(var j=0; j<r; j++) val += scl[ti+j];
			for x := left; x < left+r; x++ {
				val.incrementInt(in.At(x, y))
			}

			//for(var j=0  ; j<=r ; j++) { val += scl[ri++] - fv       ;   tcl[ti++] = Math.round(val*iarr); }
			for x := left; x <= left+r; x++ {
				val.incrementInt(in.At(rx, y))
				val.decrement(fv)
				rx++
				out.Set(tx, y, val.asColor(iarr))
				tx++
			}

			//for(var j=r+1; j<w-r; j++) { val += scl[ri++] - scl[li++];   tcl[ti++] = Math.round(val*iarr); }
			for x := left + r + 1; x < right-r; x++ {
				val.incrementInt(in.At(rx, y))
				val.decrementInt(in.At(lx, y))
				rx++
				lx++
				out.Set(tx, y, val.asColor(iarr))
				tx++
			}

			//for(var j=w-r; j<w  ; j++) { val += lv        - scl[li++];   tcl[ti++] = Math.round(val*iarr); }
			for x := right - r; x < right; x++ {
				val.increment(lv)
				val.decrementInt(in.At(lx, y))
				lx++
				out.Set(tx, y, val.asColor(iarr))
				tx++
			}
		}(y)
	}
	wg.Wait()
}

func boxBlurVertical(in image.Image, out *image.RGBA64, r int) {
	//var iarr = 1 / (r+r+1);
	var (
		iarr   = 1.0 / float64(2*r+1)
		top    = in.Bounds().Min.Y
		bottom = in.Bounds().Max.Y
		left   = in.Bounds().Min.X
		right  = in.Bounds().Max.X
	)

	wg := sync.WaitGroup{}
	//for(var i=0; i<w; i++) {
	for x := left; x < right; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			//var ti = i, li = ti, ri = ti+r*w;
			//var fv = scl[ti], lv = scl[ti+w*(h-1)], val = (r+1)*fv;
			var (
				ty  = top
				ly  = top
				ry  = top + r
				fv  = newColorVal(in.At(x, top))
				lv  = newColorVal(in.At(x, bottom-1))
				val = fv.times(float64(r + 1))
			)

			//for(var j=0; j<r; j++) val += scl[ti+j*w];
			for y := top; y < top+r; y++ {
				val.incrementInt(in.At(x, y))
			}

			//for(var j=0  ; j<=r ; j++) { val += scl[ri] - fv     ;  tcl[ti] = Math.round(val*iarr);  ri+=w; ti+=w; }
			for y := top; y <= top+r; y++ {
				val.incrementInt(in.At(x, ry))
				val.decrement(fv)
				out.Set(x, ty, val.asColor(iarr))
				ry++
				ty++
			}

			//for(var j=r+1; j<h-r; j++) { val += scl[ri] - scl[li];  tcl[ti] = Math.round(val*iarr);  li+=w; ri+=w; ti+=w; }
			for y := top + r + 1; y < bottom-r; y++ {
				val.incrementInt(in.At(x, ry))
				val.decrementInt(in.At(x, ly))
				out.Set(x, ty, val.asColor(iarr))
				ly++
				ry++
				ty++
			}

			//for(var j=h-r; j<h  ; j++) { val += lv      - scl[li];  tcl[ti] = Math.round(val*iarr);  li+=w; ti+=w; }
			for y := bottom - r; y < bottom; y++ {
				val.increment(lv)
				val.decrementInt(in.At(x, ly))
				out.Set(x, ty, val.asColor(iarr))
				ly++
				ty++
			}
		}(x)
	}
	wg.Wait()
}

func boxesForGauss(sigma float64, n int) (sizes []int) {
	wIdeal := math.Sqrt((12.0 * sigma * sigma / float64(n)) + 1.0) // Ideal averaging filter width
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2

	mIdeal := (12.0*sigma*sigma - float64(n*wl*wl+4*n*wl+3*n)) / float64(-4*wl-4)
	m := math.Round(mIdeal)
	// var sigmaActual = Math.sqrt( (m*wl*wl + (n-m)*wu*wu - n)/12 );

	sizes = make([]int, n)
	for i := range sizes {
		if float64(i) < m {
			sizes[i] = wl
		} else {
			sizes[i] = wu
		}
	}
	return
}

type colorVal struct {
	r, g, b, a float64
}

func newColorVal(c color.Color) (out *colorVal) {
	r, g, b, a := c.RGBA()
	out = &colorVal{
		r: float64(r),
		g: float64(g),
		b: float64(b),
		a: float64(a),
	}
	return
}

func (v *colorVal) times(n float64) (product *colorVal) {
	product = &colorVal{
		r: v.r * n,
		g: v.g * n,
		b: v.b * n,
		a: v.a * n,
	}
	return
}

func (v *colorVal) increment(n *colorVal) {
	v.r += n.r
	v.g += n.g
	v.b += n.b
	v.a += n.a
}

func (v *colorVal) incrementInt(n color.Color) {
	r, g, b, a := n.RGBA()
	v.r += float64(r)
	v.g += float64(g)
	v.b += float64(b)
	v.a += float64(a)
}

func (v *colorVal) decrement(n *colorVal) {
	v.r -= n.r
	v.g -= n.g
	v.b -= n.b
	v.a -= n.a
}

func (v *colorVal) decrementInt(n color.Color) {
	r, g, b, a := n.RGBA()
	v.r -= float64(r)
	v.g -= float64(g)
	v.b -= float64(b)
	v.a -= float64(a)
}

func (v *colorVal) asColor(factor float64) color.Color {
	return &color.RGBA64{
		R: uint16(math.Round(v.r * factor)),
		G: uint16(math.Round(v.g * factor)),
		B: uint16(math.Round(v.b * factor)),
		A: uint16(math.Round(v.a * factor)),
	}
}
