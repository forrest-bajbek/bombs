package utils

import (
	"bytes"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// markdown escapes raw HTML (no html.WithUnsafe), so output is safe to embed.
// Hard wraps keep single newlines visible, as the chat input expects.
var markdown = goldmark.New(
	goldmark.WithExtensions(extension.NewLinkify(
		extension.WithLinkifyAllowedProtocols([]string{"http:", "https:"}),
		extension.WithLinkifyEmailRegexp(regexp.MustCompile(`^$`)),
	), extension.Strikethrough),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		renderer.WithNodeRenderers(util.Prioritized(imageAsText{}, 100)),
	),
)

// imageAsText renders markdown images as their alt text. A real <img> would
// make every viewer's browser fetch a sender-chosen URL, leaking viewer IPs.
type imageAsText struct{}

func (imageAsText) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, func(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			w.WriteString(stdhtml.EscapeString(string(n.Text(source))))
		}
		return ast.WalkSkipChildren, nil
	})
}

// RenderMarkdown converts message text to an HTML fragment.
func RenderMarkdown(text string) string {
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(text), &buf); err != nil {
		return stdhtml.EscapeString(text)
	}
	// Output is escaped, so any literal "<a href=" is a real link.
	return strings.ReplaceAll(buf.String(), "<a href=", `<a target="_blank" rel="noopener noreferrer" href=`)
}
