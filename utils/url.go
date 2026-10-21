package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

func IsHTTPS(r *http.Request) bool {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}

func GetBaseURL(r *http.Request) string {

	domain := r.Host

	var protocol string
	if IsHTTPS(r) {
		protocol = "https"
	} else {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s", protocol, domain)
}

// urlPattern matches http(s) URLs.
// The last character cannot be common trailing punctuation, so "see https://a.com/x." keeps the period outside the match.
var urlPattern = regexp.MustCompile(`(?i)https?://[^\s<>"']*[^\s<>"'.,;:!?)\]}]`)

var urlCleaners = []func(*url.URL){
	removeQueryParams("youtu.be", "is"), // https://youtu.be/video_id?is=tracking_id
	removeQueryParams("x.com", "s"),     // https://x.com/lcamtuf/status/2112926074535612523?s=20
	removeAllQueryParams("amazon.com"),  // https://www.amazon.com/LG-27GX704A-B-Compatible-TrueBlack400-DisplayPort/dp/B0FNQ4B2Z2/?_encoding=UTF8&pd_rd_w=idnrv&content-id=amzn1.sym.24bb35ab-d728-4e6b-82e2-3dae38f5bd7c&pf_rd_p=24bb35ab-d728-4e6b-82e2-3dae38f5bd7c&pf_rd_r=TGN0NXGVNT72ZAA9PKNE&pd_rd_wg=LkWss&pd_rd_r=34f5caa9-0b09-4872-ac3e-bc990a984ba6
}

func SanitizeURLs(text string) string {
	var b strings.Builder
	last := 0
	for _, loc := range urlPattern.FindAllStringIndex(text, -1) {
		b.WriteString(text[last:loc[0]])
		b.WriteString(cleanURL(text[loc[0]:loc[1]]))
		last = loc[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	for _, clean := range urlCleaners {
		clean(u)
	}
	return u.String()
}

func hostMatches(u *url.URL, domain string) bool {
	host := strings.ToLower(u.Hostname())
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func removeQueryParams(domain string, keys ...string) func(*url.URL) {
	return func(u *url.URL) {
		if u.RawQuery == "" || !hostMatches(u, domain) {
			return
		}
		var kept []string
		for _, pair := range strings.Split(u.RawQuery, "&") {
			key, _, _ := strings.Cut(pair, "=")
			if !slices.ContainsFunc(keys, func(k string) bool { return strings.EqualFold(k, key) }) {
				kept = append(kept, pair)
			}
		}
		u.RawQuery = strings.Join(kept, "&")
	}
}

func removeAllQueryParams(domain string) func(*url.URL) {
	return func(u *url.URL) {
		if !hostMatches(u, domain) {
			return
		}
		u.RawQuery = ""
		u.ForceQuery = false
	}
}
