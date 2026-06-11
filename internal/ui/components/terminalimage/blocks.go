package terminalimage

import (
	"image"
	"image/color"
	"os"
	"strings"

	"golang.org/x/image/draw"
)

type colorMode int

const (
	colorModeNone colorMode = iota
	colorMode256
	colorModeTruecolor
)

func (m *Model) renderBlocks(box rect, img image.Image) string {
	switch detectColorMode() {
	case colorModeTruecolor:
		return m.renderImageTruecolor(box, img)
	case colorMode256:
		return m.renderImage256(box, img)
	default:
		return m.renderImageMono(box, img)
	}
}

// renderImageTruecolor draws a truecolor image inside the box using half-blocks.
func (m *Model) renderImageTruecolor(box rect, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if box.width < 1 || box.height < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := box.width
	pixelHeight := box.height * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width * 4)
		var lastFg color.RGBA
		var lastBg color.RGBA
		hasColor := false
		for x := range m.width {
			if y < box.top || y >= box.top+box.height || x < box.left || x >= box.left+box.width {
				if hasColor {
					sb.WriteString("\x1b[0m")
					hasColor = false
				}
				sb.WriteByte(' ')
				continue
			}
			ix := x - box.left
			iy := y - box.top
			top := resized[iy*2*pixelWidth+ix]
			bottom := resized[(iy*2+1)*pixelWidth+ix]
			fg := blendOnBlack(top)
			bg := blendOnBlack(bottom)

			if !hasColor || fg != lastFg || bg != lastBg {
				sb.WriteString(ansiRGB(fg, true))
				sb.WriteString(ansiRGB(bg, false))
				lastFg = fg
				lastBg = bg
				hasColor = true
			}
			sb.WriteRune('▀')
		}
		if hasColor {
			sb.WriteString("\x1b[0m")
		}
		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}

// renderImage256 draws an image using the 256-color palette and half-blocks.
func (m *Model) renderImage256(box rect, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if box.width < 1 || box.height < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := box.width
	pixelHeight := box.height * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width * 4)
		lastFg := -1
		lastBg := -1
		hasColor := false
		for x := range m.width {
			if y < box.top || y >= box.top+box.height || x < box.left || x >= box.left+box.width {
				if hasColor {
					sb.WriteString("\x1b[0m")
					hasColor = false
				}
				sb.WriteByte(' ')
				continue
			}
			ix := x - box.left
			iy := y - box.top
			top := blendOnBlack(resized[iy*2*pixelWidth+ix])
			bottom := blendOnBlack(resized[(iy*2+1)*pixelWidth+ix])
			fg := rgbTo256(top)
			bg := rgbTo256(bottom)

			if !hasColor || fg != lastFg || bg != lastBg {
				sb.WriteString(ansi256(fg, true))
				sb.WriteString(ansi256(bg, false))
				lastFg = fg
				lastBg = bg
				hasColor = true
			}
			sb.WriteRune('▀')
		}
		if hasColor {
			sb.WriteString("\x1b[0m")
		}
		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}

// renderImageMono draws a monochrome approximation without ANSI colors.
func (m *Model) renderImageMono(box rect, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if box.width < 1 || box.height < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := box.width
	pixelHeight := box.height * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	ramps := []rune{' ', '░', '▒', '▓', '█'}

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width)
		for x := range m.width {
			if y < box.top || y >= box.top+box.height || x < box.left || x >= box.left+box.width {
				sb.WriteByte(' ')
				continue
			}
			ix := x - box.left
			iy := y - box.top
			top := blendOnBlack(resized[iy*2*pixelWidth+ix])
			bottom := blendOnBlack(resized[(iy*2+1)*pixelWidth+ix])
			luma := (luminance(top) + luminance(bottom)) / 2.0
			idx := int(luma*float64(len(ramps)-1) + 0.5)
			if idx < 0 {
				idx = 0
			} else if idx >= len(ramps) {
				idx = len(ramps) - 1
			}
			sb.WriteRune(ramps[idx])
		}
		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}

func detectColorMode() colorMode {
	value := strings.ToLower(os.Getenv("COLORTERM"))
	if strings.Contains(value, "truecolor") || strings.Contains(value, "24bit") {
		return colorModeTruecolor
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "direct") {
		return colorModeTruecolor
	}
	if strings.Contains(term, "256color") {
		return colorMode256
	}
	return colorModeNone
}

func resizeImage(src image.Image, width, height int) []color.RGBA {
	if width < 1 || height < 1 {
		return nil
	}
	b := src.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW < 1 || srcH < 1 {
		return nil
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return rgbaPixels(dst)
}

func rgbaPixels(img *image.RGBA) []color.RGBA {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	width := b.Dx()
	height := b.Dy()
	if width < 1 || height < 1 {
		return nil
	}
	pixels := make([]color.RGBA, width*height)
	stride := img.Stride
	for y := range height {
		row := img.Pix[y*stride:]
		for x := range width {
			idx := x * 4
			pixels[y*width+x] = color.RGBA{
				R: row[idx],
				G: row[idx+1],
				B: row[idx+2],
				A: row[idx+3],
			}
		}
	}
	return pixels
}

func blendOnBlack(c color.RGBA) color.RGBA {
	if c.A == 255 {
		return c
	}
	if c.A == 0 {
		return color.RGBA{}
	}
	a := int(c.A)
	return color.RGBA{
		R: uint8((int(c.R) * a) / 255),
		G: uint8((int(c.G) * a) / 255),
		B: uint8((int(c.B) * a) / 255),
		A: 255,
	}
}

func ansi256(idx int, foreground bool) string {
	var sb strings.Builder
	sb.Grow(16)
	sb.WriteString("\x1b[")
	if foreground {
		sb.WriteString("38;5;")
	} else {
		sb.WriteString("48;5;")
	}
	appendInt(&sb, idx)
	sb.WriteByte('m')
	return sb.String()
}

func appendInt(sb *strings.Builder, v int) {
	if v < 0 {
		sb.WriteByte('-')
		v = -v
	}
	if v >= 100 {
		sb.WriteByte('0' + byte(v/100))
		v = v % 100
		sb.WriteByte('0' + byte(v/10))
		sb.WriteByte('0' + byte(v%10))
		return
	}
	if v >= 10 {
		sb.WriteByte('0' + byte(v/10))
		sb.WriteByte('0' + byte(v%10))
		return
	}
	sb.WriteByte('0' + byte(v))
}

func rgbTo256(c color.RGBA) int {
	r := int(c.R)
	g := int(c.G)
	b := int(c.B)
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 238 {
			return 231
		}
		return 232 + (r-8)/10
	}
	r6 := (r*5 + 127) / 255
	g6 := (g*5 + 127) / 255
	b6 := (b*5 + 127) / 255
	return 16 + 36*r6 + 6*g6 + b6
}

func luminance(c color.RGBA) float64 {
	return (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255.0
}
