package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDownloadHandler_ServesUserFilesAsAttachments(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "payload.html"), []byte("<script>alert(1)</script>"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/uploads/files/payload.html", nil)
	response := httptest.NewRecorder()
	DownloadHandler("/uploads/files/", directory).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Content-Disposition") != "attachment" {
		t.Fatalf("content disposition = %q", response.Header().Get("Content-Disposition"))
	}
	if response.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff = %q", response.Header().Get("X-Content-Type-Options"))
	}
	if !strings.Contains(response.Body.String(), "alert(1)") {
		t.Fatal("download body does not contain stored file")
	}
}
