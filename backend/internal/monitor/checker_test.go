package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckerStatusAndRedirects(t *testing.T) {
	for _, code := range []int{200, 201, 204, 301, 302, 404, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("method=%s", r.Method)
				}
				if r.URL.Path != "/" {
					t.Error("followed redirect")
				}
				w.Header().Set("Location", "/redirected")
				w.WriteHeader(code)
			}))
			defer server.Close()
			checker := NewChecker()
			checker.client.Transport = http.DefaultTransport
			result := checker.Check(context.Background(), server.URL)
			expected := "down"
			if code == 200 {
				expected = "up"
			}
			if result.Status != expected || result.HTTPStatus == nil || *result.HTTPStatus != code {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestCheckerTimeoutAndForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	checker := NewChecker()
	checker.client.Transport = http.DefaultTransport
	checker.client.Timeout = 20 * time.Millisecond
	result := checker.Check(context.Background(), server.URL)
	if result.Status != "down" || result.Error != "timeout" {
		t.Fatalf("result=%+v", result)
	}
	result = NewChecker().Check(context.Background(), server.URL)
	if result.Status != "blocked" || result.Error != "forbidden_address" {
		t.Fatalf("result=%+v", result)
	}
}

func TestCheckerTLSFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	checker := NewChecker()
	checker.client.Transport = http.DefaultTransport
	result := checker.Check(context.Background(), server.URL)
	if result.Status != "down" || result.Error != "tls" {
		t.Fatalf("result=%+v", result)
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type observedBody struct{ closed bool }

func (*observedBody) Read([]byte) (int, error) { panic("monitor must not download response body") }
func (b *observedBody) Close() error           { b.closed = true; return nil }

func TestCheckerClosesBodyWithoutReadingIt(t *testing.T) {
	body := &observedBody{}
	checker := &Checker{client: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body}, nil
	})}}
	result := checker.Check(context.Background(), "https://example.com")
	if !body.closed || result.Status != "up" {
		t.Fatalf("closed=%v result=%+v", body.closed, result)
	}
}
