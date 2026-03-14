package terminal_image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/image/draw"
)

type colorMode int

const (
	colorModeNone colorMode = iota
	colorMode256
	colorModeTruecolor
)

type Model struct {
	width         int
	height        int
	aspect        float64
	imageData     *ImageData
	fallbackLabel string
}

type ImageData struct {
	Data []byte
}

func NewModel(aspect float64, fallbackLabel string) Model {
	return Model{aspect: aspect, fallbackLabel: fallbackLabel}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetImage(pic *ImageData) {
	m.imageData = pic
}

func (m *Model) Aspect() float64 {
	return m.aspect
}

func (m *Model) View() string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	label := m.fallbackLabel
	if m.imageData != nil {
		label = "Image"
	}
	boxWidth, boxHeight, leftOffset, topOffset := m.boxRect()
	return m.renderImage(boxWidth, boxHeight, leftOffset, topOffset, m.imageData, label)
}

// renderImage decides whether to render a truecolor image or a placeholder.
func (m *Model) renderImage(boxWidth, boxHeight, leftOffset, topOffset int, pic *ImageData, label string) string {
	if pic == nil {
		return m.renderPlaceholder(label)
	}

	img, err := decode(pic)
	if err != nil {
		label = "Decode failed: " + err.Error()
		return m.renderPlaceholder(label)
	}

	mode := detectColorMode()
	switch mode {
	case colorModeTruecolor:
		return m.renderImageTruecolor(boxWidth, boxHeight, leftOffset, topOffset, img)
	case colorMode256:
		return m.renderImage256(boxWidth, boxHeight, leftOffset, topOffset, img)
	default:
		return m.renderImageMono(boxWidth, boxHeight, leftOffset, topOffset, img)
	}
}

func (m *Model) boxRect() (boxWidth, boxHeight, leftOffset, topOffset int) {
	if m.width < 1 || m.height < 1 {
		return 0, 0, 0, 0
	}
	targetWidth := max(int(float64(m.height)*m.aspect), 1)
	boxWidth = min(m.width, targetWidth)
	boxHeight = max(min(m.height, int(float64(boxWidth)/m.aspect)), 1)
	if boxWidth > m.width {
		boxWidth = m.width
	}
	targetWidth = int(float64(boxHeight) * m.aspect)
	if targetWidth > 0 && targetWidth < boxWidth {
		boxWidth = targetWidth
	}
	leftOffset = (m.width - boxWidth) / 2
	topOffset = (m.height - boxHeight) / 2
	return boxWidth, boxHeight, leftOffset, topOffset
}

// renderPlaceholder draws a label, centered.
func (m *Model) renderPlaceholder(label string) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ansi.Truncate(label, m.width, "…"))
}

// renderImageTruecolor draws a truecolor image inside the box using half-blocks.
func (m *Model) renderImageTruecolor(boxWidth, boxHeight, leftOffset, topOffset int, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if boxWidth < 1 || boxHeight < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := boxWidth
	pixelHeight := boxHeight * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width * 4)
		var lastFg color.RGBA
		var lastBg color.RGBA
		hasColor := false
		for x := range m.width {
			if y < topOffset || y >= topOffset+boxHeight || x < leftOffset || x >= leftOffset+boxWidth {
				if hasColor {
					sb.WriteString("\x1b[0m")
					hasColor = false
				}
				sb.WriteByte(' ')
				continue
			}
			ix := x - leftOffset
			iy := y - topOffset
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
func (m *Model) renderImage256(boxWidth, boxHeight, leftOffset, topOffset int, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if boxWidth < 1 || boxHeight < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := boxWidth
	pixelHeight := boxHeight * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width * 4)
		lastFg := -1
		lastBg := -1
		hasColor := false
		for x := range m.width {
			if y < topOffset || y >= topOffset+boxHeight || x < leftOffset || x >= leftOffset+boxWidth {
				if hasColor {
					sb.WriteString("\x1b[0m")
					hasColor = false
				}
				sb.WriteByte(' ')
				continue
			}
			ix := x - leftOffset
			iy := y - topOffset
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
func (m *Model) renderImageMono(boxWidth, boxHeight, leftOffset, topOffset int, img image.Image) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if boxWidth < 1 || boxHeight < 1 {
		return m.renderPlaceholder("Image")
	}

	pixelWidth := boxWidth
	pixelHeight := boxHeight * 2
	resized := resizeImage(img, pixelWidth, pixelHeight)

	ramps := []rune{' ', '░', '▒', '▓', '█'}

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width)
		for x := range m.width {
			if y < topOffset || y >= topOffset+boxHeight || x < leftOffset || x >= leftOffset+boxWidth {
				sb.WriteByte(' ')
				continue
			}
			ix := x - leftOffset
			iy := y - topOffset
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

func decode(imageData *ImageData) (image.Image, error) {
	if imageData == nil || len(imageData.Data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	img, _, err := image.Decode(bytes.NewReader(imageData.Data))
	if err != nil {
		return nil, err
	}
	return img, nil
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

func ansiRGB(c color.RGBA, foreground bool) string {
	// Avoid fmt.Sprintf in the hot path to keep rendering fast.
	var sb strings.Builder
	sb.Grow(20)
	sb.WriteString("\x1b[")
	if foreground {
		sb.WriteString("38;2;")
	} else {
		sb.WriteString("48;2;")
	}
	appendUint8(&sb, c.R)
	sb.WriteByte(';')
	appendUint8(&sb, c.G)
	sb.WriteByte(';')
	appendUint8(&sb, c.B)
	sb.WriteByte('m')
	return sb.String()
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

func appendUint8(sb *strings.Builder, v uint8) {
	if v >= 100 {
		sb.WriteByte('0' + v/100)
		v = v % 100
		sb.WriteByte('0' + v/10)
		sb.WriteByte('0' + v%10)
		return
	}
	if v >= 10 {
		sb.WriteByte('0' + v/10)
		sb.WriteByte('0' + v%10)
		return
	}
	sb.WriteByte('0' + v)
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
