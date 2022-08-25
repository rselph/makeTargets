// makeTargets project main.go
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/fogleman/gg"
)

type imageFunc func(image.Point, int) (image.Image, bool)
type renderSet struct {
	name       string
	size       image.Point
	imageFuncs []imageFunc
}

var lineCountList = []int{2, 5, 10, 30, 60, 120, 480}
var renderSets = []renderSet{
	//{"tv", image.Point{3840, 2160}, imageFuncs},
	{name: "tvx2", size: image.Point{X: 3840 * 2, Y: 2160 * 2}, imageFuncs: imageFuncs},
	//{"proj", image.Point{3840, 2400}, imageFuncs},
}

var imageFuncs = []imageFunc{
	jailWhite,
	jailBlack,
	jailDark,
	jailMid,
	jailCheck,
	check,
	radial,
	rings,
	ringFade,
	wavy,
	radialWave,
	ringWave,
	stripesh,
	stripesv,
	stripesdl,
	stripesdr,
	polkaDot,
	polkaDark,
	polkaMid,
	field,
	radialWedge,
	radialWedgeOffsetX,
	radialWedgeOffsetY,
	diamond,
	crosshatch,
	honeycomb,
	ss,
	squareWave,
}

var (
	sRGBLUT        []uint16
	inversesRGBLUT []uint16
	gamma22LUT     []uint16
	linearLUT      []uint16
)

var (
	black    = color.RGBA64{A: 65535}
	darkGray = color.RGBA64{R: 16383, G: 16383, B: 16383, A: 65535}
	midGray  = color.RGBA64{R: 32767, G: 32767, B: 32767, A: 65535}
	white    = color.RGBA64{R: 65535, G: 65535, B: 65535, A: 65535}
)

func field(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, color.Gray16{Y: uint16(n * 65535 / 480)})
	return ctx.Image(), true
}

func stripe(s image.Point, theta float64, intN int) image.Image {
	ctx, _, long := newCtx(s, white)
	ctx.Rotate(gg.Radians(theta))
	ctx.SetColor(black)

	n := float64(intN)
	f := long / n
	for x := -n; x < n; x += 2 {
		ctx.DrawRectangle(x*f, -long, f, long*2)
		ctx.Fill()
	}

	return ctx.Image()
}

func stripesdl(s image.Point, n int) (image.Image, bool) {
	return stripe(s, 45, n), false
}

func stripesdr(s image.Point, n int) (image.Image, bool) {
	return stripe(s, -45, n), false
}

func stripesv(s image.Point, n int) (image.Image, bool) {
	return stripe(s, 0, n), false
}

func stripesh(s image.Point, n int) (image.Image, bool) {
	return stripe(s, 90, n), false
}

func checkCtx(ctx *gg.Context, intN int) {
	n := float64(intN)
	var f float64
	if ctx.Width() > ctx.Height() {
		f = float64(ctx.Width()) / n
	} else {
		f = float64(ctx.Height()) / n
	}

	for y := -n / 2; y < n/2; y += 2 {
		for x := -n / 2; x < n/2; x += 2 {
			//			ctx.DrawRectangle(x*f+b.Min.X, y*f+b.Min.Y, f, f)
			ctx.DrawRectangle(x*f, y*f, f, f)
			ctx.Fill()
		}
	}
	for y := 1 - n/2; y < n/2; y += 2 {
		for x := 1 - n/2; x < n/2; x += 2 {
			ctx.DrawRectangle(x*f, y*f, f, f)
			ctx.Fill()
		}
	}
}

func check(s image.Point, intN int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, white)
	ctx.SetColor(black)

	checkCtx(ctx, intN)

	return ctx.Image(), false
}

func radial(s image.Point, numLines int) (image.Image, bool) {
	pic, b, _ := newPallete(s, nil)

	fsx := float64(s.X / 2)
	fsy := float64(s.Y / 2)
	diag := math.Sqrt(fsx*fsx + fsy*fsy)

	max := 1.0 / float64(numLines)
	max = max * math.Pi
	slope := max / diag

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		fy := float64(y)
		fy2 := fy * fy
		for x := b.Min.X; x < b.Max.X; x += 1 {
			fx := float64(x)
			r := math.Sqrt(fx*fx + fy2)
			f := r * slope
			z := math.Cos(r * f)
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func rings(s image.Point, n int) (image.Image, bool) {
	pic, b, long := newPallete(s, nil)

	f := 2.0 * math.Pi / float64(long/n)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		fy := float64(y)
		fy2 := fy * fy
		for x := b.Min.X; x < b.Max.X; x += 1 {
			fx := float64(x)
			r := math.Sqrt(fx*fx + fy2)
			z := math.Cos(r * f)
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func ringFade(s image.Point, n int) (image.Image, bool) {
	pic, b, _ := newPallete(s, nil)

	fsx := float64(s.X / 2)
	fsy := float64(s.Y / 2)
	diag := math.Sqrt(fsx*fsx + fsy*fsy)

	f := 2.0 * math.Pi / (diag / float64(n))

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		fy := float64(y)
		fy2 := fy * fy
		for x := b.Min.X; x < b.Max.X; x += 1 {
			if x == 0 && y == 0 {
				pic.Set(x, y, color.Gray{Y: 255})
			} else {
				fx := float64(x)
				r := math.Sqrt(fx*fx+fy2) * f
				z := math.Sin(r) / (r)
				pic.Set(x, y, gray(z))
			}
		}
	}

	return pic, false
}

func wavy(s image.Point, n int) (image.Image, bool) {
	pic, b, long := newPallete(s, nil)

	scale := math.Pi / (float64(long) / float64(n))
	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		cy := math.Cos(float64(y) * scale)
		for x := b.Min.X; x < b.Max.X; x += 1 {
			z := (math.Cos(float64(x)*scale) + cy) / 2.0
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func addJail(ctx *gg.Context, div float64, maxLineWidth float64) {
	width := float64(ctx.Width())
	height := float64(ctx.Height())

	lineWidth := 0.05 * width / div
	if lineWidth > maxLineWidth {
		lineWidth = maxLineWidth
	}
	ctx.SetLineWidth(lineWidth)

	ctx.DrawLine(-width, 0, width, 0)
	ctx.DrawLine(0, -height, 0, height)
	ctx.Stroke()
	for i := 1.0; true; i++ {
		d := i * width / div

		if d > width && d > height {
			break
		}

		ctx.DrawLine(-width, d, width, d)
		ctx.DrawLine(-width, -d, width, -d)
		ctx.DrawLine(d, -height, d, height)
		ctx.DrawLine(-d, -height, -d, height)
		ctx.Stroke()
	}
}

func jailCheck(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, white)

	ctx.SetColor(gray(.25))
	checkCtx(ctx, n)

	ctx.SetColor(black)
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}

func jailWhite(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, white)
	ctx.SetColor(black)
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}
func jailBlack(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, black)
	ctx.SetColor(white)
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}
func jailDark(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, darkGray)
	ctx.SetColor(white)
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}
func jailMid(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, midGray)
	ctx.SetColor(white)
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}

func diamond(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, white)
	ctx.SetColor(black)
	ctx.Rotate(gg.Radians(45))
	addJail(ctx, float64(n), 5)
	return ctx.Image(), false
}

func crosshatch(s image.Point, n int) (image.Image, bool) {
	ctx, _, _ := newCtx(s, white)
	ctx.SetColor(black)
	addJail(ctx, float64(n), 5)
	ctx.Rotate(gg.Radians(45))
	addJail(ctx, float64(n)/math.Sqrt2, 5)
	return ctx.Image(), false
}

func honeycomb(s image.Point, nInt int) (image.Image, bool) {
	ctx, b, l := newCtx(s, white)
	ctx.SetColor(black)

	n := float64(nInt)
	d := l / n
	r := d / 2
	innerR := r * math.Cos(math.Pi/6)
	side := d * math.Sin(math.Pi/6)
	wedge := side * math.Cos(math.Pi/3)

	lineWidth := 0.06 * l / n
	if lineWidth > 6 {
		lineWidth = 6
	}
	ctx.SetLineWidth(lineWidth)

	for y := b.Min.Y; y < b.Max.Y+innerR; y += 2 * innerR {
		for x := b.Min.X; x < b.Max.X+innerR; x += d + side {
			ctx.DrawRegularPolygon(6, x, y, r, 0)
			ctx.Stroke()
		}
		for x := b.Min.X + wedge + side; x < b.Max.X+innerR; x += d + side {
			ctx.DrawRegularPolygon(6, x, y+innerR, r, 0)
			ctx.Stroke()
		}
	}

	return ctx.Image(), false
}

func ss(s image.Point, nInt int) (image.Image, bool) {
	ctx, b, l := newCtx(s, white)
	ctx.SetColor(black)

	n := float64(nInt)
	lineWidth := 0.06 * l / n
	if lineWidth > 6 {
		lineWidth = 6
	}
	ctx.SetLineWidth(lineWidth)

	dx := b.Max.X / n
	dy := b.Max.Y / n
	for i := 0.0; i <= n; i++ {
		xDelta := dx * i
		yDelta := b.Max.Y - dy*i
		ctx.DrawLine(0, yDelta, xDelta, 0)
		ctx.DrawLine(xDelta, 0, 0, -yDelta)
		ctx.DrawLine(0, -yDelta, -xDelta, 0)
		ctx.DrawLine(-xDelta, 0, 0, yDelta)
		ctx.Stroke()
	}

	return ctx.Image(), false
}

func radialWedgeAngle(i, n int) float64 {
	return math.Pi*2*float64(i)/float64(n) + math.Pi/4
}
func radialWedgeImpl(s image.Point, offsetX, offsetY float64, n int) (image.Image, bool) {
	n *= 2

	ctx, b, l := newCtx(s, white)

	centerX := b.Max.X * offsetX
	centerY := b.Max.Y * offsetY
	ctx.Translate(centerX, centerY)
	r := math.Sqrt(centerX*centerX+centerY*centerY) + l

	ctx.SetColor(black)
	for i := 0; i < n; i += 2 {
		ctx.DrawArc(0, 0, r, radialWedgeAngle(i, n), radialWedgeAngle(i+1, n))
		ctx.LineTo(0, 0)
		ctx.ClosePath()
		ctx.Fill()
	}

	return ctx.Image(), false
}
func radialWedge(s image.Point, n int) (image.Image, bool) {
	return radialWedgeImpl(s, 0, 0, n)
}
func radialWedgeOffsetX(s image.Point, n int) (image.Image, bool) {
	return radialWedgeImpl(s, -1.5, 0, n)
}
func radialWedgeOffsetY(s image.Point, n int) (image.Image, bool) {
	return radialWedgeImpl(s, 0, 1.5, n)
}

func radialWave(s image.Point, n int) (image.Image, bool) {
	pic, b, _ := newPallete(s, nil)

	fn := float64(n)
	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			theta := math.Atan2(float64(y), float64(x))
			z := math.Cos(math.Pi + theta*fn)
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func squareWave(s image.Point, n int) (image.Image, bool) {
	const exp = 100.0
	pic, b, _ := newPallete(s, nil)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		zp := math.Cos(math.Pow(math.Abs(float64(y)), float64(n)/exp))
		for x := b.Min.X; x < b.Max.X; x += 1 {
			z := zp * math.Cos(math.Pow(math.Abs(float64(x)), float64(n)/100.0))
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func ringWave(s image.Point, n int) (image.Image, bool) {
	pic, b, long := newPallete(s, nil)

	f := 2.0 * math.Pi / float64(long/n)
	fn := float64(n)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		fy := float64(y)
		fy2 := fy * fy
		for x := b.Min.X; x < b.Max.X; x += 1 {
			fx := float64(x)
			r := math.Sqrt(fx*fx + fy2)

			theta := math.Atan2(fy, fx)

			z := math.Cos(math.Pi+theta*fn) * math.Cos(r*f)
			pic.Set(x, y, gray(z))
		}
	}

	return pic, true
}

func polkaDot(s image.Point, n int) (image.Image, bool) {
	return doDot(s, n, black, white)
}

func polkaDark(s image.Point, n int) (image.Image, bool) {
	return doDot(s, n, white, darkGray)
}

func polkaMid(s image.Point, n int) (image.Image, bool) {
	return doDot(s, n, white, midGray)
}

func doDot(s image.Point, n int, foreground, background color.Color) (image.Image, bool) {
	ctx, b, l := newCtx(s, background)
	ctx.SetColor(foreground)

	long := int(l)
	dotRadius := (long / n) / 5

	longMin := int(b.Min.X)
	if s.Y > s.X {
		longMin = int(b.Min.Y)
	}
	for xi := 1; xi < n; xi += 1 {
		offsetx := longMin + long*xi/n

		for yi := 1; yi < n; yi += 1 {
			offsety := longMin + long*yi/n

			ctx.DrawCircle(float64(offsetx), float64(offsety), float64(dotRadius))
			ctx.Fill()
		}
	}
	return ctx.Image(), false
}

func main() {
	var wg sync.WaitGroup

	flag.Parse()
	initLUTs()

	queue := make(chan *imageJob)
	for j := 0; j < runtime.NumCPU(); j++ {
		wg.Add(1)
		go worker(queue, &wg)
	}

	for _, set := range renderSets {
		if flag.Arg(0) != "" && set.name != flag.Arg(0) {
			continue
		}
		for _, numLines := range lineCountList {
			for _, ifunc := range set.imageFuncs {
				queue <- &imageJob{
					imageFunc: ifunc,
					imgSize:   set.size,
					numLines:  numLines,
					sizeName:  set.name,
				}
			}
		}
	}
	close(queue)

	wg.Wait()
}

type imageJob struct {
	imageFunc func(image.Point, int) (image.Image, bool)
	imgSize   image.Point
	numLines  int
	sizeName  string
}

func worker(in chan *imageJob, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range in {
		oneTask(job.imageFunc, job.imgSize, job.numLines, job.sizeName)
	}
}

func oneTask(iFunc imageFunc, imgSize image.Point, numLines int, sizeName string) {

	funcAddr := reflect.ValueOf(iFunc).Pointer()
	funcName := runtime.FuncForPC(funcAddr).Name()
	if i := strings.LastIndex(funcName, "."); i >= 0 {
		funcName = funcName[i+1:]
	}

	img, shouldClamp := iFunc(imgSize, numLines)
	//img, _ := imageFunc(imgSize, numLines)
	if img == nil {
		return
	}

	fileName := makeName(sizeName, funcName, numLines)

	srgbImg := srgbConvert(img, sRGBLUT)
	fmt.Println(fileName)
	save(srgbImg, fileName)

	if shouldClamp {
		clamp := clamp(img)
		fileName += "_clamp"
		fmt.Println(fileName)
		save(clamp, fileName)
	}
}

func makeName(sizeName, typeName string, numLines int) string {
	fileName := fmt.Sprintf("%s_%s_%03d", sizeName, typeName, numLines)
	return fileName
}

func newPallete(s image.Point, background color.Color) (pic *image.RGBA64, b image.Rectangle, l int) {
	b = image.Rect(-s.X/2, -s.Y/2, s.X/2, s.Y/2)
	pic = image.NewRGBA64(b)
	if s.X > s.Y {
		l = s.X
	} else {
		l = s.Y
	}

	if background != nil {
		draw.Draw(pic, b, &image.Uniform{C: background}, image.Point{}, draw.Src)
	}

	return
}

type floatPoint struct {
	X float64
	Y float64
}
type floatRect struct {
	Min floatPoint
	Max floatPoint
}

func rect(minx, miny, maxx, maxy float64) floatRect {
	return floatRect{
		Min: floatPoint{X: minx, Y: miny},
		Max: floatPoint{X: maxx, Y: maxy},
	}
}

func newCtx(s image.Point, background color.Color) (ctx *gg.Context, b floatRect, l float64) {
	sx := float64(s.X)
	sy := float64(s.Y)
	b = rect(-sx/2, -sy/2, sx/2, sy/2)

	if sx > sy {
		l = sx
	} else {
		l = sy
	}

	ctx = gg.NewContext(s.X, s.Y)

	if background != nil {
		ctx.SetColor(background)
		ctx.DrawRectangle(0, 0, sx, sy)
		ctx.Fill()
	}

	ctx.Translate(sx/2, sy/2)

	return
}

func save(i image.Image, name string) {
	w, err := os.Create(name + ".png")
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	err = png.Encode(w, i)
	if err != nil {
		log.Fatal(err)
	}
}

func gray(z float64) color.Color {
	return color.Gray16{Y: uint16((z + 1.0) * 32767.5)}
}

func clamp(in image.Image) image.Image {
	b := in.Bounds()
	out := image.NewRGBA64(b)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			r, _, _, _ := in.At(x, y).RGBA()
			if r > 0x7fff {
				out.SetRGBA64(x, y, white)
			} else {
				out.SetRGBA64(x, y, black)
			}
		}
	}

	return gaussianBlur(out, 1.0)
}

func srgbConvert(in image.Image, lut []uint16) image.Image {
	b := in.Bounds()
	out := image.NewRGBA64(b)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			r, g, b, a := in.At(x, y).RGBA()
			out.SetRGBA64(x, y, color.RGBA64{R: lut[r], G: lut[g], B: lut[b], A: uint16(a)})
		}
	}

	return out
}

func initLUTs() {
	a := 0.055
	e := 1.0 / 2.4
	a1 := a + 1.0
	var csrgb, cl float64

	sRGBLUT = make([]uint16, 65536)
	for i := range sRGBLUT {
		cl = float64(i) / 65535.0
		if cl <= 0.0031308 {
			csrgb = cl * 12.92
		} else {
			csrgb = a1*math.Pow(cl, e) - a
		}
		sRGBLUT[i] = uint16(csrgb * 65535.0)
	}

	inversesRGBLUT = make([]uint16, 65536)
	for i := range inversesRGBLUT {
		csrgb = float64(i) / 65535.0
		if csrgb <= 0.04045 {
			cl = csrgb / 12.92
		} else {
			cl = math.Pow((csrgb+a)/a1, 2.4)
		}
		inversesRGBLUT[i] = uint16(cl * 65535.0)
	}

	gamma := 2.2
	gamma22LUT = make([]uint16, 65536)
	for i := range gamma22LUT {
		cl = float64(i) / 65535.0
		gamma22LUT[i] = uint16(math.Pow(cl, gamma) * 65535.0)
	}

	linearLUT = make([]uint16, 65536)
	for i := range linearLUT {
		linearLUT[i] = uint16(i)
	}
}
