package main

import (
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

func main() {
	var size int
	var out string
	var pngPath string
	flag.IntVar(&size, "size", 256, "icon size in pixels")
	flag.StringVar(&out, "out", "tmus.png", "output file")
	flag.StringVar(&pngPath, "png", "", "input png file")
	flag.Parse()

	if size <= 0 {
		panic("size must be positive")
	}
	if pngPath == "" {
		panic("missing -png")
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	if err := renderPNG(img, pngPath); err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func renderPNG(img *image.RGBA, pngPath string) error {
	f, err := os.Open(pngPath)
	if err != nil {
		return err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	xdraw.CatmullRom.Scale(img, img.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return nil
}
