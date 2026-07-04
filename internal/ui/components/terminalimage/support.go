package terminalimage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

type rect struct {
	width  int
	height int
	left   int
	top    int
}

func (m *Model) boxRect() rect {
	if m.width < 1 || m.height < 1 {
		return rect{}
	}
	targetWidth := max(int(float64(m.height)*m.aspect), 1)
	boxWidth := min(m.width, targetWidth)
	boxHeight := max(min(m.height, int(float64(boxWidth)/m.aspect)), 1)
	if boxWidth > m.width {
		boxWidth = m.width
	}
	targetWidth = int(float64(boxHeight) * m.aspect)
	if targetWidth > 0 && targetWidth < boxWidth {
		boxWidth = targetWidth
	}
	return rect{
		width:  boxWidth,
		height: boxHeight,
		left:   (m.width - boxWidth) / 2,
		top:    (m.height - boxHeight) / 2,
	}
}

// renderPlaceholder draws a label, centered.
func (m *Model) renderPlaceholder(label string) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	truncateRight := truncate.Right{}.MaxWidth(m.width)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, truncateRight.Render(sanitize.TerminalText(label)))
}

func decode(imageData *Data) (image.Image, error) {
	if imageData == nil || len(imageData.Bytes) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	img, _, err := image.Decode(bytes.NewReader(imageData.Bytes))
	if err != nil {
		return nil, err
	}
	return img, nil
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
