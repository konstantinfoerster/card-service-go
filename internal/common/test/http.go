package commontest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/stretchr/testify/require"
)

const CookieEncryptionKey = "01234567890123456789012345678901"

type httpRequest struct {
	url     string
	method  string
	body    []byte
	cookies []*http.Cookie
	header  map[string]string
}

func NewRequest(options ...func(*httpRequest)) *http.Request {
	r := &httpRequest{
		header:  make(map[string]string),
		cookies: make([]*http.Cookie, 0),
	}
	for _, opt := range options {
		opt(r)
	}

	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}

	req := httptest.NewRequest(r.method, r.url, body)

	for _, c := range r.cookies {
		req.AddCookie(c)
	}

	for k, v := range r.header {
		req.Header.Set(k, v)
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
		req.body = body
	}
}

func WithJSONBody(t *testing.T, v interface{}) func(*httpRequest) {
	raw := ToJSON(t, v)

	return func(req *httpRequest) {
		req.header[fiber.HeaderContentType] = fiber.MIMEApplicationJSON

		req.body = raw
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

func WithEncryptedCookie(t *testing.T, cookie http.Cookie) func(*httpRequest) {
	return func(req *httpRequest) {
		v, err := encryptcookie.EncryptCookie(cookie.Value, CookieEncryptionKey)
		require.NoError(t, err)

		cookie.Value = v

		req.cookies = append(req.cookies, &cookie)
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
