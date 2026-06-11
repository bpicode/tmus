package terminalimage

import (
	"bytes"
	"hash/fnv"
	"image/color"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
)

const kittyImageNamespace = 0x6d0000

func (m *Model) renderKitty(box rect, imageID int) string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if box.width < 1 || box.height < 1 {
		return m.renderPlaceholder("Image")
	}

	lines := make([]string, m.height)
	for y := range m.height {
		var sb strings.Builder
		sb.Grow(m.width * 5)
		for x := range m.width {
			if y < box.top || y >= box.top+box.height || x < box.left || x >= box.left+box.width {
				sb.WriteByte(' ')
				continue
			}
			sb.WriteString(kittyPlaceholder(imageID, x-box.left, y-box.top))
		}
		sb.WriteString("\x1b[0m")
		lines[y] = sb.String()
	}
	return strings.Join(lines, "\n")
}

func kittyPlaceholder(imageID, column, row int) string {
	c := color.RGBA{
		R: uint8(imageID >> 16),
		G: uint8(imageID >> 8),
		B: uint8(imageID),
		A: 255,
	}
	var sb strings.Builder
	sb.WriteString(ansiRGB(c, true))
	sb.WriteRune(kitty.Placeholder)
	sb.WriteRune(kitty.Diacritic(row))
	sb.WriteRune(kitty.Diacritic(column))
	return sb.String()
}

func (m *Model) uploadKitty() string {
	if m.width < 1 || m.height < 1 || m.pic == nil || m.imageID == 0 {
		return ""
	}
	img, err := decode(m.pic)
	if err != nil {
		return ""
	}
	box := m.boxRect()
	if box.width < 1 || box.height < 1 {
		return ""
	}
	var buf bytes.Buffer
	err = kitty.EncodeGraphics(&buf, img, &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Format:           kitty.PNG,
		Transmission:     kitty.Direct,
		ID:               m.imageID,
		Columns:          box.width,
		Rows:             box.height,
		VirtualPlacement: true,
		Quite:            2,
		Chunk:            true,
	})
	if err != nil {
		return ""
	}
	m.uploadedID = m.imageID
	return buf.String()
}

func (m *Model) deleteKitty(imageID int) string {
	if imageID == 0 {
		return ""
	}
	var buf bytes.Buffer
	err := kitty.EncodeGraphics(&buf, nil, &kitty.Options{
		Action:          kitty.Delete,
		ID:              imageID,
		Delete:          kitty.DeleteID,
		DeleteResources: true,
		Quite:           2,
	})
	if err != nil {
		return ""
	}
	if m.uploadedID == imageID {
		m.uploadedID = 0
	}
	return buf.String()
}

func detectKittySupport() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if os.Getenv("KONSOLE_VERSION") != "" {
		return true
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "WezTerm", "ghostty", "Ghostty", "iTerm.app":
		return true
	default:
		return false
	}
}

func kittyImageID(data []byte) int {
	h := fnv.New32a()
	_, _ = h.Write(data)
	return kittyImageNamespace | int(h.Sum32()&0xffff)
}
