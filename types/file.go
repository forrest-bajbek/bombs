package types

import (
	"strconv"
	"time"

	"github.com/forrest-bajbek/bombs/routes"
)

const (
	MaxFileBytes       = 10 << 20 // per photo, after downscaling
	MaxFilesPerMessage = 10
	MaxRequestBytes    = MaxFilesPerMessage*MaxFileBytes + (1 << 20) // + text and multipart overhead

	// Uploads are downscaled to fit within this box before storage.
	MaxImageDimension = 1600

	// Checked against DecodeConfig before a full decode, so a small file
	// that expands into an enormous bitmap is rejected before it's
	// allocated.
	MaxImagePixels = 50_000_000

	// Every stored byte lives in the WASM SQLite heap for the lifetime of
	// the process. main.go raises that heap's ceiling; this keeps us from
	// walking into it, so a full disk surfaces as a rejected upload
	// instead of SQLITE_NOMEM breaking every query in the app.
	StorageBudgetBytes = 512 << 20

	// Attachment grids render at most this many tiles; the last becomes
	// the "+n" tile when there are more.
	AttachmentTilesShown = 4
)

// AllowedImageMimeTypes is the whitelist for both storage and serving.
// SVG is excluded on purpose: it's a script-capable document format, and
// serving one inline from this origin would be stored XSS against the
// origin holding the auth cookie.
var AllowedImageMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// NewFile is an upload in flight: decoded, downscaled and sniffed, but not
// yet encrypted or stored.
type NewFile struct {
	MimeType string
	Content  []byte
}

// MessageFile is attachment metadata only. It deliberately carries no
// content - it's loaded for whole pages of message history at a time, and
// the bytes are fetched one at a time over ChatFile instead.
type MessageFile struct {
	FileID   int    `json:"file_id"`
	ChatID   int    `json:"chat_id"`
	MimeType string `json:"mime_type"`
	Position int    `json:"position"`
}

func (f *MessageFile) FileURL() string {
	return routes.URL(routes.ChatFile, strconv.Itoa(f.ChatID), strconv.Itoa(f.FileID))
}

// File is a single attachment's decrypted bytes, for the serving route.
type File struct {
	ID        int
	CreatedAt time.Time
	MimeType  string
	Content   []byte
}
