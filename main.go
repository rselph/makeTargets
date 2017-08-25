// makeTargets project main.go
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/fogleman/gg"
	"golang.org/x/image/tiff"
)

var lineCountList = []int{2, 5, 10, 30, 60, 120, 480}
var sizeList = []image.Point{
	//	image.Point{X: 3840, Y: 2160},
	image.Point{X: 3840, Y: 2400},
}
var sizeNames = []string{
	//	"tv",
	"proj",
}
var imageFuncs = []func(image.Point, int) (image.Image, bool){
	jailWhite,
	jailBlack,
	jailDark,
	jailMid,
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
}

var sRGBLUT []uint16
var inversesRGBLUT []uint16
var linearLUT []uint16

var black = color.RGBA64{0, 0, 0, 65535}
var darkGray = color.RGBA64{16383, 16383, 16383, 65535}
var midGray = color.RGBA64{32767, 32767, 32767, 65535}
var white = color.RGBA64{65535, 65535, 65535, 65535}

func stripe(s image.Point, Θ float64, n int) image.Image {
	pic, b, long := newPallete(s, white)

	Θ = Θ * math.Pi / 180.0
	slope := math.Tan(Θ)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		fy := float64(y)
		for x := b.Min.X; x < b.Max.X; x += 1 {
			b := fy - slope*float64(x)
			xbit := int(b) * n / long
			if b < 0 {
				xbit = ^xbit
			}
			if xbit&1 != 0 {
				pic.Set(x, y, black)
			}
		}
	}

	return pic
}

func stripesdl(s image.Point, n int) (image.Image, bool) {
	return stripe(s, 45, n), false
}

func stripesdr(s image.Point, n int) (image.Image, bool) {
	return stripe(s, -45, n), false
}

func field(s image.Point, n int) (image.Image, bool) {
	pic, _, _ := newPallete(s, color.Gray16{uint16(n * 65535 / 480)})
	return pic, true
}

func stripesv(s image.Point, n int) (image.Image, bool) {
	pic, b, _ := newPallete(s, nil)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			xbit := x * n / b.Max.X
			if x < 0 {
				xbit = ^xbit
			}
			pic.Set(x, y, color.Gray{uint8(255 * (xbit & 1))})
		}
	}

	// Take care of single pixel areas on the edge.
	checkFix(pic)

	return pic, false
}

func stripesh(s image.Point, n int) (image.Image, bool) {
	pic, b, _ := newPallete(s, white)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		ybit := (y * n / b.Max.Y)
		if y < 0 {
			ybit = ^ybit
		}
		if ybit&1 != 0 {
			for x := b.Min.X; x < b.Max.X; x += 1 {
				pic.Set(x, y, black)
			}
		}
	}

	// Take care of single pixel areas on the edge.
	checkFix(pic)

	return pic, false
}

func check(s image.Point, n int) (image.Image, bool) {
	pic, b, long := newPallete(s, nil)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		ybit := (y * n / long)
		if y < 0 {
			ybit = ^ybit
		}
		for x := b.Min.X; x < b.Max.X; x += 1 {
			xbit := x * n / long
			if x < 0 {
				xbit = ^xbit
			}
			pic.Set(x, y, color.Gray{uint8(255 * ((xbit ^ ybit) & 1))})
		}
	}

	// Take care of single pixel areas on the edge.
	checkFix(pic)

	return pic, false
}

func checkFix(pic *image.RGBA64) {
	b := pic.Bounds()
	topLeft := pic.At(b.Min.X+1, b.Min.Y+1)
	topRight := pic.At(b.Max.X-1, b.Min.Y+1)
	bottomLeft := pic.At(b.Min.X+1, b.Max.Y-1)
	bottomRight := pic.At(b.Max.X-1, b.Max.Y-1)

	left := pic.At(b.Min.X, b.Min.Y+1) != topLeft || pic.At(b.Min.X, b.Max.Y-1) != bottomLeft
	right := pic.At(b.Max.X, b.Min.Y+1) != topRight || pic.At(b.Max.X, b.Max.Y-1) != bottomRight
	top := pic.At(b.Min.X+1, b.Min.Y) != topLeft || pic.At(b.Max.X-1, b.Min.Y) != topRight
	bottom := pic.At(b.Min.X+1, b.Max.Y) != bottomLeft || pic.At(b.Max.X-1, b.Max.Y) != bottomRight

	if left {
		for y := b.Min.Y; y < b.Max.Y; y += 1 {
			pic.Set(b.Min.X, y, pic.At(b.Min.X+1, y))
		}
	}

	if right {
		for y := b.Min.Y; y < b.Max.Y; y += 1 {
			pic.Set(b.Max.X, y, pic.At(b.Max.X-1, y))
		}
	}

	if top {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			pic.Set(x, b.Min.Y, pic.At(x, b.Min.Y+1))
		}
	}

	if bottom {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			pic.Set(x, b.Max.Y, pic.At(x, b.Max.Y-1))
		}
	}
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
				pic.Set(x, y, color.Gray{255})
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
			Θ := math.Atan2(float64(y), float64(x))
			z := math.Cos(math.Pi + Θ*fn)
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

			Θ := math.Atan2(fy, fx)

			z := math.Cos(math.Pi+Θ*fn) * math.Cos(r*f)
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

	initsRGBLUT()

	queue := make(chan *imageJob)
	for j := 0; j < runtime.NumCPU(); j++ {
		wg.Add(1)
		go worker(queue, &wg)
	}

	for i, sizeName := range sizeNames {
		for _, numLines := range lineCountList {
			for _, imageFunc := range imageFuncs {
				queue <- &imageJob{
					imageFunc: imageFunc,
					imgSize:   sizeList[i],
					numLines:  numLines,
					sizeName:  sizeName,
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

func oneTask(imageFunc func(image.Point, int) (image.Image, bool), imgSize image.Point, numLines int, sizeName string) {

	funcAddr := reflect.ValueOf(imageFunc).Pointer()
	funcName := runtime.FuncForPC(funcAddr).Name()
	if i := strings.LastIndex(funcName, "."); i >= 0 {
		funcName = funcName[i+1:]
	}

	//	img, shouldDither := imageFunc(imgSize, numLines)
	img, _ := imageFunc(imgSize, numLines)
	if img == nil {
		return
	}

	fileName := makeName(sizeName, funcName, imgSize, numLines)
	//	if shouldDither {
	//		wg.Add(1)
	//		go func(img image.Image, fileName string) {
	//			defer wg.Done()
	//			dith := ditherize(img)
	//			fileName += "_dith"
	//			fmt.Println(fileName)
	//			save(dith, fileName)
	//		}(img, fileName)
	//	}

	srgbImg := srgbConvert(img, linearLUT)
	fmt.Println(fileName)
	save(srgbImg, fileName)

}

func makeName(sizeName, typeName string, imgSize image.Point, numLines int) string {
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
		draw.Draw(pic, b, &image.Uniform{background}, image.ZP, draw.Src)
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
	w, err := os.Create(name + ".tiff")
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	err = tiff.Encode(w, i, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
	if err != nil {
		log.Fatal(err)
	}
}

func gray(z float64) color.Color {
	return color.Gray16{uint16((z + 1.0) * 32767.5)}
}

func ditherize(in image.Image) image.Image {
	b := in.Bounds()
	out := image.NewRGBA64(b)
	myRand := rand.New(rand.NewSource(42))

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			r, _, _, _ := in.At(x, y).RGBA()
			if myRand.Uint32()&0xFFFF < r {
				out.SetRGBA64(x, y, white)
			} else {
				out.SetRGBA64(x, y, black)
			}
		}
	}

	return out
}

func srgbConvert(in image.Image, lut []uint16) image.Image {
	b := in.Bounds()
	out := image.NewRGBA64(b)

	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			r, g, b, a := in.At(x, y).RGBA()
			out.SetRGBA64(x, y, color.RGBA64{lut[r], lut[g], lut[b], uint16(a)})
		}
	}

	return out
}

func initsRGBLUT() {
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

	linearLUT = make([]uint16, 65536)
	for i := range linearLUT {
		linearLUT[i] = uint16(i)
	}
}
