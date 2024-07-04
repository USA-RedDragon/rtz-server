package utils

import (
	"io"
	"net/http"
	"time"
)

var client = http.Client{
	Timeout: 5 * time.Second,
}

func HTTPRequest(method string, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
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
