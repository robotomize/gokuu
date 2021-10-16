package httputil

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "gokuu/0.0.0"

var ErrStatusCode = errors.New("http status != 200")

// DefaultSourceHTTPClient return preconfigured HTTP client
func DefaultSourceHTTPClient() SourceHTTPClient {
	return SourceHTTPClient{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          20000,
				MaxIdleConnsPerHost:   1000,
				DisableCompression:    true,
				IdleConnTimeout:       5 * time.Minute,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
			},
		},
	}
}

// NewHTTPClient return prepared SourceHTTPClient
func NewHTTPClient(client *http.Client) SourceHTTPClient {
	return SourceHTTPClient{client: client}
}

type SourceHTTPClient struct {
	client *http.Client
}

func (f SourceHTTPClient) UserAgent() string {
	return defaultUserAgent
}

// Get implements HTTP method GET client and returns the slice byte from the body
func (f SourceHTTPClient) Get(ctx context.Context, u url.URL) ([]byte, error) {
	return f.fetch(ctx, u)
}

func (f SourceHTTPClient) fetch(ctx context.Context, u url.URL) ([]byte, error) {
	req, err := f.prepareRequest(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("build HTTP request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make HTTP request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, %s: %w", resp.StatusCode, resp.Status, ErrStatusCode)
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	contentType := resp.Header.Get("Content-Type")
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch {
	case strings.Contains(contentType, "zip"), strings.Contains(contentType, "application/x-gzip"), strings.Contains(contentEncoding, "gzip"):
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable create gzip.NewReader: %w", err)
		}
		reader = gz
		defer reader.Close()

	default:
		reader = resp.Body
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("read body: %w", err)
		}
	}

	return b, nil
}

func (f SourceHTTPClient) prepareRequest(ctx context.Context, u url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept-Encoding", "gzip")

	return req, nil
}
