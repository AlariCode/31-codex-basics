package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORS_AllowsConfiguredOrigin(t *testing.T) {
	handler := WithCORS("http://localhost:3000", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusOK) }))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("unexpected CORS response: status=%d headers=%v", response.Code, response.Header())
	}
}

func TestWithCORS_RejectsUnknownOriginPreflight(t *testing.T) {
	handler := WithCORS("http://localhost:3000", http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://example.invalid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}
