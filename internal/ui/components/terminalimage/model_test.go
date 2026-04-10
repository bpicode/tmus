package terminalimage

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	smallPNGBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(smallPNGBase64))
	data, err := io.ReadAll(d)
	assert.NoError(t, err)

	m := NewModel(2.0, "My Fallback")
	m.SetSize(10, 10)
	m.SetImage(&Data{Bytes: data})
	view := m.View()
	assert.NotContains(t, view, "My Fallback")
	assert.NotContains(t, view, "Decode failed")
}

func TestModelFallback(t *testing.T) {
	m := NewModel(2.0, "My Fallback")
	m.SetSize(20, 20)
	m.SetImage(nil)
	assert.Contains(t, m.View(), "My Fallback")
}

func TestModelError(t *testing.T) {
	m := NewModel(2.0, "My Fallback")
	m.SetSize(20, 20)
	m.SetImage(&Data{Bytes: []byte("invalid-image-data")})
	assert.Contains(t, m.View(), "Decode failed")
}
