package utils

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"github.com/forrest-bajbek/bombs/types"

	// jpeg and png are imported above, which registers their decoders.
	_ "image/gif"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// jpegQuality balances the re-encode against the fact that every stored
// byte occupies the in-memory database for the life of the process.
const jpegQuality = 85

var errUnsupportedImage = errors.New("Only JPEG, PNG, GIF and WebP images are allowed.")

// ProcessUpload validates raw upload bytes and returns what should actually
// be stored. Images larger than types.MaxImageDimension are downscaled and
// re-encoded; everything else is stored byte-for-byte as uploaded.
//
// The returned MIME type is always sniffed from the final bytes. The
// client's multipart Content-Type header and the uploaded filename are
// both ignored - neither is trustworthy, and the sniffed value is what
// ends up in the Content-Type of the response that serves it back.
func ProcessUpload(raw []byte) (types.NewFile, error) {
	if len(raw) == 0 {
		return types.NewFile{}, errors.New("Empty file.")
	}
	if len(raw) > types.MaxFileBytes {
		return types.NewFile{}, fmt.Errorf("Photos must be smaller than %dMB.", types.MaxFileBytes>>20)
	}

	// Read just the header first. A decompression bomb is small on the
	// wire and enormous in memory, so the dimensions have to be checked
	// before the pixels are ever allocated.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return types.NewFile{}, errUnsupportedImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > types.MaxImagePixels {
		return types.NewFile{}, errors.New("Image dimensions are too large.")
	}

	// GIFs are stored exactly as uploaded. image/gif decodes only the
	// first frame, so re-encoding one would silently throw away the
	// animation - better to keep the original and let the size cap apply.
	if format == "gif" {
		return finalize(raw)
	}

	if cfg.Width <= types.MaxImageDimension && cfg.Height <= types.MaxImageDimension {
		return finalize(raw)
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return types.NewFile{}, errUnsupportedImage
	}

	dst := downscale(src)

	var buf bytes.Buffer
	// JPEG has no alpha channel, so anything transparent would composite
	// to black. Those go out as PNG instead, at the cost of a larger file.
	if hasAlpha(dst) {
		err = png.Encode(&buf, dst)
	} else {
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality})
	}
	if err != nil {
		return types.NewFile{}, err
	}

	return finalize(buf.Bytes())
}

func finalize(content []byte) (types.NewFile, error) {
	if len(content) > types.MaxFileBytes {
		return types.NewFile{}, fmt.Errorf("Photos must be smaller than %dMB.", types.MaxFileBytes>>20)
	}
	mimeType := http.DetectContentType(content)
	if !types.AllowedImageMimeTypes[mimeType] {
		return types.NewFile{}, errUnsupportedImage
	}
	return types.NewFile{MimeType: mimeType, Content: content}, nil
}

// downscale resizes src to fit inside a MaxImageDimension box, preserving
// aspect ratio.
func downscale(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	scale := float64(types.MaxImageDimension) / float64(w)
	if s := float64(types.MaxImageDimension) / float64(h); s < scale {
		scale = s
	}

	w = int(float64(w) * scale)
	h = int(float64(h) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// hasAlpha reports whether img contains any non-opaque pixel, so the
// encoder can pick a format that won't discard it.
func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return true
			}
		}
	}
	return false
}
