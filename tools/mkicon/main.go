// Command mkicon renders the SimDiag application icon.
//
// The mark echoes what the tool produces: a control point with callout labels
// running off it, which is what every generated diagram looks like. Each size is
// drawn at 4x and box-filtered down, so it stays legible at 16 px rather than
// relying on an automatic resize of one large image.
//
// Usage:
//
//	go run ./tools/mkicon build/windows/icon.ico winres
//
// The first argument is the .ico to write. The optional second argument is a
// directory that also receives one PNG per size, which is what go-winres
// consumes to build the embedded resource. See make resources.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Colours match the application shell so the icon and the window agree.
var (
	background = color.NRGBA{R: 0x14, G: 0x17, B: 0x1c, A: 0xff}
	accent     = color.NRGBA{R: 0x4d, G: 0x9d, B: 0xe0, A: 0xff}
	labelInk   = color.NRGBA{R: 0xe4, G: 0xe8, B: 0xef, A: 0xff}
)

// iconSizes are the sizes stored in the .ico, from the taskbar's 16 px up.
var iconSizes = []int{16, 24, 32, 48, 64, 128, 256}

const supersample = 4

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mkicon <icon.ico> [png-output-dir]")
		os.Exit(1)
	}

	if err := run(os.Args[1], pngDir()); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
}

// pngDir is the optional directory receiving one PNG per size.
func pngDir() string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	return ""
}

func run(icoPath, pngPath string) error {
	payloads := make([][]byte, 0, len(iconSizes))

	for _, size := range iconSizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(size)); err != nil {
			return err
		}
		payloads = append(payloads, buf.Bytes())

		if pngPath != "" {
			name := fmt.Sprintf("%s/icon%d.png", pngPath, size)
			if err := os.WriteFile(name, buf.Bytes(), 0644); err != nil {
				return err
			}
		}
	}

	return os.WriteFile(icoPath, encodeICO(payloads), 0644)
}

// render draws the icon at size*supersample and box-filters it down.
func render(size int) *image.NRGBA {
	big := size * supersample
	canvas := image.NewNRGBA(image.Rect(0, 0, big, big))

	unit := float64(big) / 64 // the design grid is 64x64

	fillRoundedRect(canvas, 0, 0, float64(big), float64(big), 12*unit, background)

	// The control point.
	dotX, dotY, dotR := 17*unit, 32*unit, 6.5*unit
	fillCircle(canvas, dotX, dotY, dotR, accent)

	// Three callout labels, each joined to the point by a leader line.
	for _, l := range []struct{ y, width float64 }{{15, 26}, {32, 22}, {49, 26}} {
		y, x := l.y*unit, 30*unit
		w, h := l.width*unit, 7*unit

		drawLine(canvas, dotX+dotR*0.6, dotY, x, y, 1.6*unit, accent)
		strokeRoundedRect(canvas, x, y-h/2, w, h, 2*unit, 1.6*unit, labelInk)
	}

	return downsample(canvas, size)
}

func fillRoundedRect(dst *image.NRGBA, x, y, w, h, r float64, c color.NRGBA) {
	for py := int(y); py < int(y+h); py++ {
		for px := int(x); px < int(x+w); px++ {
			if insideRoundedRect(float64(px)+0.5, float64(py)+0.5, x, y, w, h, r) {
				setIfInside(dst, px, py, c)
			}
		}
	}
}

func strokeRoundedRect(dst *image.NRGBA, x, y, w, h, r, thickness float64, c color.NRGBA) {
	for py := int(y - thickness); py < int(y+h+thickness); py++ {
		for px := int(x - thickness); px < int(x+w+thickness); px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			outer := insideRoundedRect(fx, fy, x, y, w, h, r)
			inner := insideRoundedRect(fx, fy, x+thickness, y+thickness,
				w-2*thickness, h-2*thickness, math.Max(0, r-thickness))
			if outer && !inner {
				setIfInside(dst, px, py, c)
			}
		}
	}
}

func insideRoundedRect(px, py, x, y, w, h, r float64) bool {
	if px < x || py < y || px > x+w || py > y+h {
		return false
	}

	cx := math.Min(math.Max(px, x+r), x+w-r)
	cy := math.Min(math.Max(py, y+r), y+h-r)

	return math.Hypot(px-cx, py-cy) <= r ||
		(px >= x+r && px <= x+w-r) ||
		(py >= y+r && py <= y+h-r)
}

func fillCircle(dst *image.NRGBA, cx, cy, r float64, c color.NRGBA) {
	for py := int(cy - r - 1); py <= int(cy+r+1); py++ {
		for px := int(cx - r - 1); px <= int(cx+r+1); px++ {
			if math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy) <= r {
				setIfInside(dst, px, py, c)
			}
		}
	}
}

func drawLine(dst *image.NRGBA, x1, y1, x2, y2, thickness float64, c color.NRGBA) {
	steps := int(math.Hypot(x2-x1, y2-y1) * 2)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		fillCircle(dst, x1+(x2-x1)*t, y1+(y2-y1)*t, thickness/2, c)
	}
}

func setIfInside(dst *image.NRGBA, x, y int, c color.NRGBA) {
	if image.Pt(x, y).In(dst.Bounds()) {
		dst.SetNRGBA(x, y, c)
	}
}

// downsample box-filters the supersampled canvas, which is what gives the mark
// clean edges at small sizes.
func downsample(src *image.NRGBA, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))

	for y := range size {
		for x := range size {
			var r, g, b, a int
			for sy := range supersample {
				for sx := range supersample {
					p := src.NRGBAAt(x*supersample+sx, y*supersample+sy)
					r += int(p.R) * int(p.A)
					g += int(p.G) * int(p.A)
					b += int(p.B) * int(p.A)
					a += int(p.A)
				}
			}

			if a == 0 {
				dst.SetNRGBA(x, y, color.NRGBA{})
				continue
			}

			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / a),
				G: uint8(g / a),
				B: uint8(b / a),
				A: uint8(a / (supersample * supersample)),
			})
		}
	}

	return dst
}

// encodeICO wraps PNG payloads in an ICO container. Writes to a bytes.Buffer
// cannot fail, so their errors are deliberately not checked.
func encodeICO(payloads [][]byte) []byte {
	var out bytes.Buffer

	write := func(v any) {
		_ = binary.Write(&out, binary.LittleEndian, v)
	}

	// ICONDIR: reserved, type 1 (icon), image count.
	write(uint16(0))
	write(uint16(1))
	write(uint16(len(payloads)))

	offset := 6 + 16*len(payloads)
	for i, size := range iconSizes {
		dim := byte(size)
		if size >= 256 {
			dim = 0 // 0 means 256 in the ICO directory
		}

		write(dim)        // width
		write(dim)        // height
		write(byte(0))    // palette size
		write(byte(0))    // reserved
		write(uint16(1))  // colour planes
		write(uint16(32)) // bits per pixel
		write(uint32(len(payloads[i])))
		write(uint32(offset))

		offset += len(payloads[i])
	}

	for _, payload := range payloads {
		out.Write(payload)
	}

	return out.Bytes()
}
