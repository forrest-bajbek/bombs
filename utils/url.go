package utils

import (
	"fmt"
	"net/http"
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
