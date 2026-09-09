package monitor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"

	"uptime-backend/internal/publichttp"
)

// CheckResult contains a single observation, never a response body or an unsafe error message.
type CheckResult struct {
	CheckedAt  time.Time
	Status     string
	HTTPStatus *int
	Error      string
}

// Checker performs bounded GET checks against public destinations.
type Checker struct{ client *http.Client }

// NewChecker creates the production checker; only an immediate 200 is successful.
func NewChecker() *Checker { return &Checker{client: publichttp.NewClient(10 * time.Second)} }

// Check reads response headers and closes the body without downloading the page.
func (c *Checker) Check(ctx context.Context, target string) CheckResult {
	result := CheckResult{Status: "down"}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err == nil {
		request.Header.Set("User-Agent", "Uptime-Monitor/1.0")
		var response *http.Response
		response, err = c.client.Do(request)
		if response != nil {
			response.Body.Close()
			code := response.StatusCode
			result.HTTPStatus = &code
			if code == http.StatusOK {
				result.Status = "up"
			}
		}
	}
	if err != nil {
		result.Error = checkError(err)
		if result.Error == "forbidden_address" {
			result.Status = "blocked"
		}
	}
	result.CheckedAt = time.Now().UTC()
	return result
}

func checkError(err error) string {
	var networkError net.Error
	var dnsError *net.DNSError
	var certificateError *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	switch {
	case errors.Is(err, publichttp.ErrForbidden):
		return "forbidden_address"
	case errors.As(err, &networkError) && networkError.Timeout():
		return "timeout"
	case errors.As(err, &dnsError):
		return "dns"
	case errors.As(err, &certificateError), errors.As(err, &unknownAuthority):
		return "tls"
	default:
		return "network"
	}
}
