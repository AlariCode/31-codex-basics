package publichttp

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestAllowedHost(t *testing.T) {
	for _, host := range []string{"localhost", "LOCALHOST.", "a.localhost", "127.0.0.1", "10.1.1.1", "169.254.169.254", "100.100.100.200", "192.168.1.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1%lo0", "64:ff9b::7f00:1", "2002:7f00:1::1", "224.0.0.1"} {
		if AllowedHost(host) {
			t.Errorf("allowed %s", host)
		}
	}
	for _, host := range []string{"example.com", "8.8.8.8", "2606:4700:4700::1111"} {
		if !AllowedHost(host) {
			t.Errorf("blocked %s", host)
		}
	}
}

func TestDialChecksAllAnswersAndPinsPublicIP(t *testing.T) {
	answers := []netip.Addr{netip.MustParseAddr("8.8.8.8")}
	dialed := ""
	expected := errors.New("test dial")
	dialer := safeDialer{
		lookup: func(context.Context, string, string) ([]netip.Addr, error) { return answers, nil },
		dial: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialed = address
			return nil, expected
		},
	}
	_, err := dialer.connect(context.Background(), "tcp", "example.com:443")
	if !errors.Is(err, expected) || dialed != "8.8.8.8:443" {
		t.Fatalf("dial=%s err=%v", dialed, err)
	}
	// A later private DNS answer must be rejected, including a mixed public/private answer set.
	answers = append(answers, netip.MustParseAddr("127.0.0.1"))
	dialed = ""
	_, err = dialer.connect(context.Background(), "tcp", "example.com:443")
	if !errors.Is(err, ErrForbidden) || dialed != "" {
		t.Fatalf("dial=%s err=%v", dialed, err)
	}
}

func TestClientBlocksLocalRequests(t *testing.T) {
	client := NewClient(time.Second)
	_, err := client.Get("http://127.0.0.1:8080")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
