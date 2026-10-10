//go:build ignore

// icon draws the original, neutral app icon of OpenDiablo2.app: a dark rounded
// square with a crimson ring and two pale vertical bars (a stylised "II").
// It uses no Blizzard artwork. Usage: go run scripts/icon/main.go out.png [size]
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		os.Stderr.WriteString("usage: icon out.png [size]\n")
		os.Exit(2)
	}

	size := 1024

	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n >= 16 {
			size = n
		}
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	c := s / 2
	// macOS icon grid: the artwork fills 80% of the canvas
	half := s * 0.40
	radius := s * 0.18

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5-c, float64(y)+0.5-c
			// signed distance to a rounded square
			qx, qy := math.Abs(px)-(half-radius), math.Abs(py)-(half-radius)
			d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - radius
			a := clamp(0.5 - d)

			if a <= 0 {
				continue
			}

			// background: vertical gradient from near-black to dark red
			t := (py + half) / (2 * half)
			col := mix(color.NRGBA{18, 14, 16, 255}, color.NRGBA{60, 12, 14, 255}, t)

			// ring
			r := math.Hypot(px, py)
			ringR, ringW := half*0.62, half*0.07
			col = mix(col, color.NRGBA{190, 32, 36, 255}, clamp(ringW/2+0.5-math.Abs(r-ringR)))

			// two bars
			for _, bx := range []float64{-half * 0.14, half * 0.14} {
				bd := math.Max(math.Abs(px-bx)-half*0.055, math.Abs(py)-half*0.30)
				col = mix(col, color.NRGBA{232, 222, 205, 255}, clamp(0.5-bd))
				// serifs at both ends make it read as the numeral II, not "pause"
				for _, sy := range []float64{-half * 0.30, half * 0.30} {
					sd := math.Max(math.Abs(px-bx)-half*0.125, math.Abs(py-sy)-half*0.035)
					col = mix(col, color.NRGBA{232, 222, 205, 255}, clamp(0.5-sd))
				}
			}

			col.A = uint8(a * 255)
			img.SetNRGBA(x, y, col)
		}
	}

	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func clamp(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	t = clamp(t)
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }

	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}
