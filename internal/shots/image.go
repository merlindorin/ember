package shots

import (
	"image"
	"image/color"
	stddraw "image/draw"

	"golang.org/x/image/draw"
)

// crop is img's part r, in its pixels.
func crop(img image.Image, r image.Rectangle) image.Image {
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	stddraw.Draw(out, out.Bounds(), img, r.Min.Add(img.Bounds().Min), stddraw.Src)
	return out
}

// halve averages each 2×2 block of img, from pixels to points.
func halve(img image.Image) image.Image {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/2, b.Dy()/2))
	for y := range b.Dy() / 2 {
		for x := range b.Dx() / 2 {
			var s [4]int
			for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				c, _ := color.NRGBAModel.Convert(img.At(b.Min.X+2*x+d[0], b.Min.Y+2*y+d[1])).(color.NRGBA)
				s[0] += int(c.R)
				s[1] += int(c.G)
				s[2] += int(c.B)
				s[3] += int(c.A)
			}
			out.SetNRGBA(
				x,
				y,
				color.NRGBA{mean(s[0]), mean(s[1]), mean(s[2]), mean(s[3])},
			)
		}
	}
	return out
}

// resize is img at w×h: halved exactly when that is half its size, else
// through Catmull-Rom.
func resize(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	if b.Dx() == 2*w && b.Dy() == 2*h {
		return halve(img)
	}
	if b.Dx() == w && b.Dy() == h {
		return img
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Src, nil)
	return out
}

// hstack lines imgs up left to right, top-aligned.
func hstack(imgs []image.Image) image.Image {
	w, h := 0, 0
	for _, img := range imgs {
		w += img.Bounds().Dx()
		h = max(h, img.Bounds().Dy())
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	x := 0
	for _, img := range imgs {
		b := img.Bounds()
		stddraw.Draw(out, image.Rect(x, 0, x+b.Dx(), b.Dy()), img, b.Min, stddraw.Src)
		x += b.Dx()
	}
	return out
}

// mean is a channel's average over the four pixels summed in sum.
func mean(sum int) uint8 {
	v := (sum + 2) / 4
	if v < 0 || v > 255 {
		return 255
	}
	return uint8(v)
}
