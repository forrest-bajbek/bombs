package utils

import (
	"net/url"
	"testing"
)

func TestSanitizeURLs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no urls", "hello world", "hello world"},
		{"youtu.be param removed", "https://youtu.be/abc123?is=tracking", "https://youtu.be/abc123"},
		{"x param removed", "https://x.com/u/status/2?s=20", "https://x.com/u/status/2"},
		{"x keeps other params", "https://x.com/u/status/2?s=20&t=z", "https://x.com/u/status/2?t=z"},
		{"amazon params removed", "https://www.amazon.com/dp/B0?tag=aff&ref=abc&th=1", "https://www.amazon.com/dp/B0?th=1"},
		{"subdomain matches", "https://www.x.com/u?s=20", "https://www.x.com/u"},
		{"key match is case-insensitive", "https://x.com/u?S=20", "https://x.com/u"},
		{"other domain untouched", "https://example.com/p?s=keep&is=keep", "https://example.com/p?s=keep&is=keep"},
		{"lookalike domain untouched", "https://notx.com/?s=keep", "https://notx.com/?s=keep"},
		{"param on wrong domain untouched", "https://x.com/u?tag=keep", "https://x.com/u?tag=keep"},
		{"fragment kept", "https://x.com/u?s=20#top", "https://x.com/u#top"},
		{"remaining params keep order", "https://x.com/u?z=1&s=20&a=2", "https://x.com/u?z=1&a=2"},
		{"trailing period stays in text", "see https://x.com/u?s=20.", "see https://x.com/u."},
		{"parentheses stay in text", "(https://x.com/u/status/2?s=20)", "(https://x.com/u/status/2)"},
		{"http scheme", "http://x.com/a?s=1", "http://x.com/a"},
		{"uppercase scheme matches", "HTTPS://x.com/a?s=1", "https://x.com/a"},
		{
			"multiple urls and text",
			"a https://youtu.be/v?is=abc. b (https://x.com/u?s=20&t=z) c https://example.com/p?s=keep",
			"a https://youtu.be/v. b (https://x.com/u?t=z) c https://example.com/p?s=keep",
		},
		{"non-http scheme ignored", "ftp://x.com/a?s=keep", "ftp://x.com/a?s=keep"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeURLs(tt.in); got != tt.want {
				t.Errorf("SanitizeURLs(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveAllQueryParams(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		in     string
		want   string
	}{
		{"all params removed", "example.com", "https://example.com/p?a=1&b=2", "https://example.com/p"},
		{"subdomain matches", "example.com", "https://www.example.com/p?a=1", "https://www.example.com/p"},
		{"fragment kept", "example.com", "https://example.com/p?a=1#top", "https://example.com/p#top"},
		{"empty query marker removed", "example.com", "https://example.com/p?", "https://example.com/p"},
		{"no query unchanged", "example.com", "https://example.com/p", "https://example.com/p"},
		{"other domain untouched", "example.com", "https://other.com/p?a=1", "https://other.com/p?a=1"},
		{"lookalike domain untouched", "example.com", "https://notexample.com/p?a=1", "https://notexample.com/p?a=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			removeAllQueryParams(tt.domain)(u)
			if got := u.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
