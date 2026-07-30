package monitor

import "testing"

func TestValidURL_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, value := range []string{"https://example.com", "http://localhost:8080/health"} {
		if !validURL(value) {
			t.Errorf("validURL(%q) = false", value)
		}
	}
}

func TestValidURL_RejectsMissingSchemeOrHost(t *testing.T) {
	for _, value := range []string{"example.com", "ftp://example.com", "https://"} {
		if validURL(value) {
			t.Errorf("validURL(%q) = true", value)
		}
	}
}
