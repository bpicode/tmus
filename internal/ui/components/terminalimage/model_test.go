package terminalimage

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel(t *testing.T) {
	data := smallPNGData(t)

	m := NewModel(2.0, "My Fallback", RendererBlocks)
	m.SetSize(10, 10)
	m.SetImage(&Data{Bytes: data})
	view := m.View()
	assert.NotContains(t, view, "My Fallback")
	assert.NotContains(t, view, "Decode failed")
}

func TestModelFallback(t *testing.T) {
	m := NewModel(2.0, "My Fallback", RendererBlocks)
	m.SetSize(20, 20)
	m.SetImage(nil)
	assert.Contains(t, m.View(), "My Fallback")
}

func TestModelError(t *testing.T) {
	m := NewModel(2.0, "My Fallback", RendererBlocks)
	m.SetSize(20, 20)
	m.SetImage(&Data{Bytes: []byte("invalid-image-data")})
	assert.Contains(t, m.View(), "Decode failed")
}

func TestModelNoneRenderer(t *testing.T) {
	m := NewModel(2.0, "My Fallback", RendererNone)
	m.SetSize(20, 20)
	m.SetImage(&Data{Bytes: []byte("invalid-image-data")})
	assert.Contains(t, m.View(), "My Fallback")
}

func TestModelKittyRenderer(t *testing.T) {
	data := smallPNGData(t)

	m := NewModel(2.0, "My Fallback", RendererKitty)
	m.SetSize(10, 10)
	raw := m.SetImage(&Data{Bytes: data})
	assert.Contains(t, raw, "\x1b_G")

	view := m.View()
	assert.Contains(t, view, string(kittyPlaceholder(kittyImageID(data), 0, 0)))
	assert.NotContains(t, view, "My Fallback")

	raw = m.Clear()
	assert.Contains(t, raw, "\x1b_G")
	assert.Contains(t, raw, "a=d")
}

func TestModelKittyRendererDeletesWhenSizeCollapses(t *testing.T) {
	data := smallPNGData(t)

	m := NewModel(2.0, "My Fallback", RendererKitty)
	m.SetSize(10, 10)
	require.NotEmpty(t, m.SetImage(&Data{Bytes: data}))

	raw := m.SetSize(0, 10)
	assert.Contains(t, raw, "\x1b_G")
	assert.Contains(t, raw, "a=d")
}

func TestModelKittyRendererReplacesImageOnResize(t *testing.T) {
	data := smallPNGData(t)

	m := NewModel(2.0, "My Fallback", RendererKitty)
	m.SetSize(10, 10)
	require.NotEmpty(t, m.SetImage(&Data{Bytes: data}))

	raw := m.SetSize(8, 8)
	assert.Contains(t, raw, "a=d")
	assert.Contains(t, raw, "a=T")
	assert.GreaterOrEqual(t, strings.Count(raw, "\x1b_G"), 2)
}

func TestKittyPlaceholderUsesRowThenColumnDiacritics(t *testing.T) {
	placeholder := kittyPlaceholder(42, 1, 0)
	want := string([]rune{kitty.Placeholder, kitty.Diacritic(0), kitty.Diacritic(1)})
	assert.Contains(t, placeholder, want)
}

func TestAutoRendererDetectsKittySupport(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")

	m := NewModel(2.0, "My Fallback", RendererAuto)
	m.SetSize(20, 10)
	raw := m.SetImage(&Data{Bytes: []byte("invalid-image-data")})
	assert.Empty(t, raw)
	assert.Contains(t, m.View(), "Decode failed")
}

func smallPNGData(t *testing.T) []byte {
	t.Helper()
	smallPNGBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(smallPNGBase64))
	data, err := io.ReadAll(d)
	require.NoError(t, err)
	return data
}
