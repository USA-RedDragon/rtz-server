package utils

import (
	"context"
	"io"
	"net/http"
	"time"
)

//nolint:golint,gochecknoglobals
var client = http.Client{
	Timeout: 5 * time.Second,
}

func HTTPRequest(ctx context.Context, method string, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &http.Response{}, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	return resp, nil
}
