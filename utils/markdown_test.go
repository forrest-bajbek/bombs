package utils

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		contains []string
		excludes []string
	}{
		{"bold", "**b**", []string{"<strong>b</strong>"}, nil},
		{"italic", "*i*", []string{"<em>i</em>"}, nil},
		{"code", "`c`", []string{"<code>c</code>"}, nil},
		{"strikethrough", "~~s~~", []string{"<del>s</del>"}, nil},
		{"raw html escaped", "<script>alert(1)</script>", []string{"raw HTML omitted"}, []string{"<script>"}},
		{"bare url linkified", "see https://example.com/x", []string{`href="https://example.com/x"`, `target="_blank"`, `rel="noopener noreferrer"`}, nil},
		{"http url linkified", "http://example.com", []string{`href="http://example.com"`}, nil},
		{"ftp not linkified", "ftp://example.com", nil, []string{"<a "}},
		{"email not linkified", "a@b.com", nil, []string{"<a "}},
		{"javascript link dropped", "[x](javascript:alert(1))", nil, []string{"javascript:"}},
		{"newline becomes br", "a\nb", []string{"<br>"}, nil},
		{"image becomes alt text", "![alt](http://x/y.png)", []string{"alt"}, []string{"<img", "http://x/y.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown(tt.in)
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("RenderMarkdown(%q) = %q, want it to contain %q", tt.in, got, s)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(got, s) {
					t.Errorf("RenderMarkdown(%q) = %q, want it not to contain %q", tt.in, got, s)
				}
			}
		})
	}
}
