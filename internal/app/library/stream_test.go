package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		wantURL  string
		wantErr  bool
	}{
		{
			name:     "Valid URL file standard",
			filename: "test1.url",
			content:  "[InternetShortcut]\nURL=https://somafm.com/groovesalad.pls\n",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantErr:  false,
		},
		{
			name:     "Valid URL file case-insensitive",
			filename: "test2.url",
			content:  "[internetshortcut]\nurl=https://somafm.com/groovesalad.pls\n",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantErr:  false,
		},
		{
			name:     "URL file without internet shortcut section header (lenient)",
			filename: "test3.url",
			content:  "URL=https://somafm.com/groovesalad.pls\n",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantErr:  false,
		},
		{
			name:     "Valid stream file standard",
			filename: "test4.stream",
			content:  "https://somafm.com/groovesalad.pls\n",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantErr:  false,
		},
		{
			name:     "Stream file with comments and empty lines",
			filename: "test5.stream",
			content:  "# SomaFM Groove Salad\n// Alternative comment\n\nhttps://somafm.com/groovesalad.pls\n",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantErr:  false,
		},
		{
			name:     "Empty stream file",
			filename: "test6.stream",
			content:  "\n",
			wantURL:  "",
			wantErr:  true,
		},
		{
			name:     "Invalid URL file empty",
			filename: "test7.url",
			content:  "",
			wantURL:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(path, []byte(tt.content), 0644)
			require.NoError(t, err)

			gotURL, err := ParseStreamFile(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantURL, gotURL)
			}
		})
	}
}

func TestResolvePlayable(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid stream file
	streamPath := filepath.Join(tempDir, "SomaFM Groove Salad.stream")
	err := os.WriteFile(streamPath, []byte("https://somafm.com/groovesalad.pls\n"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		wantURL  string
		wantName string
		wantOk   bool
	}{
		{
			name:     "Remote path",
			path:     "https://somafm.com/groovesalad.pls",
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantName: "groovesalad.pls",
			wantOk:   true,
		},
		{
			name:     "Supported local audio file",
			path:     "/home/user/song.mp3",
			wantURL:  "/home/user/song.mp3",
			wantName: "song.mp3",
			wantOk:   true,
		},
		{
			name:     "Valid stream file path",
			path:     streamPath,
			wantURL:  "https://somafm.com/groovesalad.pls",
			wantName: "SomaFM Groove Salad",
			wantOk:   true,
		},
		{
			name:     "Non-existent stream file",
			path:     filepath.Join(tempDir, "nonexistent.stream"),
			wantURL:  "",
			wantName: "",
			wantOk:   false,
		},
		{
			name:     "Unsupported document file",
			path:     "/home/user/document.txt",
			wantURL:  "",
			wantName: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotName, gotOk := ResolvePlayable(tt.path)
			assert.Equal(t, tt.wantURL, gotURL)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
