// Package executor sends HTTP requests using net/http and returns responses.
// It has no knowledge of YAML, terminal output, or file formats.
package executor

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apitool/apitool/internal/request"
)

// NetworkError wraps errors that represent a failure to reach the server:
// connection refused, DNS failure, timeout, etc.
// The caller maps these to exit code 2.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string { return e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }

// Send executes the given Request and returns a Response.
//
// A non-nil error means the request could not be sent at all.
// Wrap type: *NetworkError for transport failures, plain error for bad input.
// HTTP error status codes (4xx, 5xx) are not errors — they are valid responses.
//
// timeout is the maximum time to wait for a response. Zero means no timeout.
func Send(req *request.Request, timeout time.Duration) (*request.Response, error) {
	targetURL, err := buildURL(req.URL, req.Params)
	if err != nil {
		return nil, err // bad URL is a tool error, not a network error
	}

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := http.DefaultClient
	if timeout != 0 {
		client = &http.Client{Timeout: timeout}
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Err: fmt.Errorf("reading response body: %w", err)}
	}

	return &request.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		Headers:    map[string][]string(resp.Header),
		Body:       body,
		Duration:   elapsed,
	}, nil
}

// buildURL appends query params to rawURL using proper URL encoding.
func buildURL(rawURL string, params map[string]string) (string, error) {
	if len(params) == 0 {
		return rawURL, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}
