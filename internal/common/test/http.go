package commontest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type httpRequest struct {
	url     string
	method  string
	body    *[]byte
	cookies []*http.Cookie
}

func NewRequest(options ...func(*httpRequest)) *http.Request {
	r := &httpRequest{}
	for _, opt := range options {
		opt(r)
	}

	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(*r.body)
	}

	req := httptest.NewRequest(r.method, r.url, body)

	for _, c := range r.cookies {
		req.AddCookie(c)
	}

	return req
}

func WithURL(url string) func(*httpRequest) {
	return func(req *httpRequest) {
		req.url = url
	}
}

func WithMethod(m string) func(*httpRequest) {
	return func(req *httpRequest) {
		req.method = m
	}
}

func WithBody(body []byte) func(*httpRequest) {
	return func(req *httpRequest) {
		req.body = &body
	}
}

func WithCookie(cookie *http.Cookie) func(*httpRequest) {
	return func(req *httpRequest) {
		if cookie == nil {
			return
		}

		req.cookies = append(req.cookies, cookie)
	}
}

func Close(t *testing.T, resp *http.Response) {
	t.Helper()

	err := resp.Body.Close()
	if err != nil {
		t.Logf("failed to close response body %v", err)

		return
	}
}
