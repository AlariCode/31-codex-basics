package monitor

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestFaviconFetcherFetch_PrefersLinkedPNGAndCachesIt(t *testing.T) {
	linkedPNG := testPNG(t, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	fallbackPNG := testPNG(t, color.RGBA{R: 50, G: 60, B: 70, A: 255})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			_, _ = writer.Write([]byte(`<html><head><link rel="icon" type="image/png" href="/brand.png"></head></html>`))
		case "/brand.png":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(linkedPNG)
		case "/favicon.png":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(fallbackPNG)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	directory := t.TempDir()
	monitorID := uuid.New()
	path, err := NewFaviconFetcher(directory).Fetch(context.Background(), server.URL, monitorID)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if path != "/uploads/favicons/"+monitorID.String()+".png" {
		t.Fatalf("path = %q", path)
	}
	stored, err := os.ReadFile(filepath.Join(directory, monitorID.String()+".png"))
	if err != nil || !bytes.Equal(stored, linkedPNG) {
		t.Fatalf("stored=%q err=%v", stored, err)
	}
}

func TestFaviconFetcherFetch_RemovesObsoleteIconWhenNoPNGExists(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	directory := t.TempDir()
	monitorID := uuid.New()
	previousPath := filepath.Join(directory, monitorID.String()+".png")
	if err := os.WriteFile(previousPath, []byte("old icon"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewFaviconFetcher(directory).Fetch(context.Background(), server.URL, monitorID)
	if err == nil {
		t.Fatal("Fetch() error = nil")
	}
	if _, statErr := os.Stat(previousPath); !os.IsNotExist(statErr) {
		t.Fatalf("obsolete favicon remains, stat error = %v", statErr)
	}
}

func testPNG(t *testing.T, pixel color.Color) []byte {
	t.Helper()
	image := image.NewRGBA(image.Rect(0, 0, 1, 1))
	image.Set(0, 0, pixel)
	var result bytes.Buffer
	if err := png.Encode(&result, image); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}
