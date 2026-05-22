package library

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURLShortcut(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantURL string
		wantErr bool
	}{
		{
			name:    "standard",
			content: "[InternetShortcut]\nURL=https://somafm.com/groovesalad.pls\n",
			wantURL: "https://somafm.com/groovesalad.pls",
		},
		{
			name:    "case-insensitive",
			content: "[internetshortcut]\nurl=https://somafm.com/groovesalad.pls\n",
			wantURL: "https://somafm.com/groovesalad.pls",
		},
		{
			name:    "without section header",
			content: "URL=https://somafm.com/groovesalad.pls\n",
			wantURL: "https://somafm.com/groovesalad.pls",
		},
		{
			name:    "empty",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := parseURLShortcut(strings.NewReader(tt.content))
			if tt.wantErr {
				assert.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURL, gotURL)
			}
		})
	}
}

func TestParseStreamShortcut(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantURL string
		wantErr bool
	}{
		{
			name:    "standard",
			content: "https://somafm.com/groovesalad.pls\n",
			wantURL: "https://somafm.com/groovesalad.pls",
		},
		{
			name:    "comments and empty lines",
			content: "# SomaFM Groove Salad\n// Alternative comment\n\nhttps://somafm.com/groovesalad.pls\n",
			wantURL: "https://somafm.com/groovesalad.pls",
		},
		{
			name:    "empty",
			content: "\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := parseStreamShortcut(strings.NewReader(tt.content))
			if tt.wantErr {
				assert.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURL, gotURL)
			}
		})
	}
}
