// jina fetches a URL via Jina's Reader service and returns the result
// as clean markdown. Intended as a BIN tool inside an agent sandbox.
//
// Usage:
//
//	jina https://example.com/article
//	# → markdown on stdout
//
// Optional env:
//
//	JINA_API_KEY  — raises rate limits above the anonymous free tier
//	JINA_ENGINE   — scraping engine override: browser | direct | readerlm
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/openotters/bin/internal/cli"
)

const (
	readerEndpoint = "https://r.jina.ai/"
	requestTimeout = 30 * time.Second
	maxBodyBytes   = 256 * 1024
	errSnippetMax  = 400
)

func main() {
	cli.Run(fetchMarkdown)
}

// fetchMarkdown sends args (a single URL) to Jina's Reader endpoint
// and returns the response body as markdown. Extracted from main to
// keep cyclomatic complexity per function below the project's cap.
func fetchMarkdown(args string) (string, error) {
	target := strings.TrimSpace(args)
	if target == "" {
		return "", fmt.Errorf("usage: jina <url>")
	}

	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid url %q: must be http(s) with a host", target)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := buildRequest(ctx, target)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{Timeout: requestTimeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", target, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", httpError(resp.StatusCode, body)
	}

	return strings.TrimRight(string(body), "\n"), nil
}

// buildRequest builds the Jina Reader GET request with the markdown
// content-negotiation headers and any auth/engine env overrides.
func buildRequest(ctx context.Context, target string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readerEndpoint+target, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-Return-Format", "markdown")

	if key := os.Getenv("JINA_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	if eng := os.Getenv("JINA_ENGINE"); eng != "" {
		req.Header.Set("X-Engine", eng)
	}

	return req, nil
}

// httpError formats a non-2xx response into a single-line error,
// truncating the body at errSnippetMax to keep the message readable.
func httpError(status int, body []byte) error {
	snippet := body
	if len(snippet) > errSnippetMax {
		snippet = snippet[:errSnippetMax]
	}

	return fmt.Errorf("jina %d: %s", status, strings.TrimSpace(string(snippet)))
}
