package terminalimage

type Renderer string

const (
	RendererAuto   Renderer = "auto"
	RendererKitty  Renderer = "kitty"
	RendererBlocks Renderer = "blocks"
	RendererNone   Renderer = "none"
)

type Model struct {
	width      int
	height     int
	aspect     float64
	pic        *Data
	fallback   string
	renderer   Renderer
	imageID    int
	uploadedID int
}

type Data struct {
	Bytes []byte
}

func NewModel(aspect float64, fallback string, renderer Renderer) Model {
	if renderer == "" {
		renderer = RendererAuto
	}
	return Model{aspect: aspect, fallback: fallback, renderer: renderer}
}

func (m *Model) SetSize(width, height int) string {
	if m.width == width && m.height == height {
		return ""
	}
	oldUploadedID := m.uploadedID
	m.width = width
	m.height = height
	if width < 1 || height < 1 {
		return m.deleteKitty(oldUploadedID)
	}
	if m.pic == nil || !m.usesKitty() {
		return ""
	}
	return m.deleteKitty(oldUploadedID) + m.uploadKitty()
}

func (m *Model) SetImage(pic *Data) string {
	oldID := m.imageID
	m.pic = pic
	if pic == nil {
		m.imageID = 0
		return m.deleteKitty(oldID)
	}
	m.imageID = kittyImageID(pic.Bytes)
	if !m.usesKitty() {
		return m.deleteKitty(oldID)
	}
	if oldID != 0 && oldID != m.imageID {
		return m.deleteKitty(oldID) + m.uploadKitty()
	}
	if m.uploadedID == m.imageID {
		return ""
	}
	return m.uploadKitty()
}

func (m *Model) Clear() string {
	m.pic = nil
	m.imageID = 0
	return m.deleteKitty(m.uploadedID)
}

func (m *Model) Aspect() float64 {
	return m.aspect
}

func (m *Model) View() string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	if m.pic == nil {
		return m.renderPlaceholder(m.fallback)
	}
	if m.effectiveRenderer() == RendererNone {
		return m.renderPlaceholder(m.fallback)
	}

	img, err := decode(m.pic)
	if err != nil {
		return m.renderPlaceholder("Decode failed: " + err.Error())
	}
	box := m.boxRect()
	switch m.effectiveRenderer() {
	case RendererKitty:
		if m.imageID == 0 {
			return m.renderPlaceholder("Image")
		}
		return m.renderKitty(box, m.imageID)
	}
	return m.renderBlocks(box, img)
}

func (m *Model) effectiveRenderer() Renderer {
	switch m.renderer {
	case RendererKitty, RendererBlocks, RendererNone:
		return m.renderer
	case RendererAuto:
		if detectKittySupport() {
			return RendererKitty
		}
		return RendererBlocks
	default:
		return RendererBlocks
	}
}

func (m *Model) usesKitty() bool {
	return m.effectiveRenderer() == RendererKitty
}
