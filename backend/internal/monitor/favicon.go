package monitor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/html"
	"uptime-backend/internal/publichttp"
)

const (
	maxFaviconSize = 1 << 20
	maxHTMLSize    = 512 << 10
)

var errFaviconNotFound = errors.New("PNG favicon not found")

// FaviconResolver downloads a PNG favicon and returns its public path.
type FaviconResolver interface {
	Fetch(context.Context, string, uuid.UUID) (string, error)
}

// FaviconFetcher finds and locally caches PNG favicons so the browser never needs to request a monitored site directly.
type FaviconFetcher struct {
	directory string
	client    *http.Client
}

// NewFaviconFetcher creates a favicon resolver that stores images in directory.
func NewFaviconFetcher(directory string) *FaviconFetcher {
	client := publichttp.NewClient(5 * time.Second)
	client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("too many redirects")
		}
		return nil
	}
	return &FaviconFetcher{directory: directory, client: client}
}

// Fetch downloads the site's PNG favicon and returns its local public URL.
func (fetcher *FaviconFetcher) Fetch(ctx context.Context, siteURL string, monitorID uuid.UUID) (string, error) {
	requestContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filename := monitorID.String() + ".png"
	path := filepath.Join(fetcher.directory, filename)
	candidates := faviconCandidates(requestContext, fetcher.client, siteURL)
	for _, candidate := range candidates {
		image, err := fetchPNG(requestContext, fetcher.client, candidate)
		if err != nil {
			continue
		}
		if err := fetcher.store(filename, image); err != nil {
			return "", fmt.Errorf("store favicon: %w", err)
		}
		return "/uploads/favicons/" + filename, nil
	}

	_ = os.Remove(path)
	return "", errFaviconNotFound
}

func faviconCandidates(ctx context.Context, client *http.Client, siteURL string) []string {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return []string{}
	}
	root := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: "/favicon.png"}
	candidates := []string{root.String()}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, siteURL, nil)
	if err != nil {
		return candidates
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	response, err := client.Do(request)
	if err != nil {
		return candidates
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return candidates
	}

	return append(faviconLinks(response.Body, response.Request.URL), candidates...)
}

func faviconLinks(body io.Reader, pageURL *url.URL) []string {
	tokenizer := html.NewTokenizer(io.LimitReader(body, maxHTMLSize))
	pngLinks := []string{}
	otherLinks := []string{}
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		token := tokenizer.Token()
		if token.Data != "link" {
			continue
		}
		attributes := map[string]string{}
		for _, attribute := range token.Attr {
			attributes[strings.ToLower(attribute.Key)] = attribute.Val
		}
		if !strings.Contains(strings.ToLower(attributes["rel"]), "icon") {
			continue
		}
		candidate, err := pageURL.Parse(attributes["href"])
		if err != nil || candidate.Scheme != "http" && candidate.Scheme != "https" || candidate.Host == "" || candidate.User != nil {
			continue
		}
		if strings.EqualFold(attributes["type"], "image/png") || strings.HasSuffix(strings.ToLower(candidate.Path), ".png") {
			pngLinks = appendUnique(pngLinks, candidate.String())
			continue
		}
		otherLinks = appendUnique(otherLinks, candidate.String())
	}
	return append(pngLinks, otherLinks...)
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func fetchPNG(ctx context.Context, client *http.Client, imageURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "image/png")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected favicon status %d", response.StatusCode)
	}
	image, err := io.ReadAll(io.LimitReader(response.Body, maxFaviconSize+1))
	if err != nil || len(image) > maxFaviconSize {
		return nil, errors.New("invalid favicon size")
	}
	configuration, err := png.DecodeConfig(bytes.NewReader(image))
	if err != nil || configuration.Width > 1024 || configuration.Height > 1024 {
		return nil, errors.New("invalid PNG favicon")
	}
	return image, nil
}

func (fetcher *FaviconFetcher) store(filename string, image []byte) error {
	if err := os.MkdirAll(fetcher.directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(fetcher.directory, ".favicon-*.png")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	if _, err := temporary.Write(image); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, filepath.Join(fetcher.directory, filename)); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	return nil
}
