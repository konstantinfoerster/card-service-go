package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/stretchr/testify/require"
)

const CookieEncryptionKey = "01234567890123456789012345678901"

type RequestOpt func(*httpRequest)

type httpRequest struct {
	header  map[string]string
	url     string
	method  string
	body    []byte
	cookies []*http.Cookie
}

func NewRequest(options ...RequestOpt) *http.Request {
	r := &httpRequest{
		header:  make(map[string]string),
		cookies: make([]*http.Cookie, 0),
	}
	for _, opt := range options {
		if opt == nil {
			continue
		}

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

func WithURL(url string) RequestOpt {
	return func(req *httpRequest) {
		req.url = url
	}
}

func WithMethod(m string) RequestOpt {
	return func(req *httpRequest) {
		req.method = m
	}
}

func WithBody(body []byte) RequestOpt {
	return func(req *httpRequest) {
		req.body = body
	}
}

func WithMultipartFile(t *testing.T, f io.Reader, name string) RequestOpt {
	return func(req *httpRequest) {
		body := new(bytes.Buffer)
		mw := multipart.NewWriter(body)
		w, err := mw.CreateFormFile("file", name)
		require.NoError(t, err)

		_, err = io.Copy(w, f)
		require.NoError(t, err)
		err = mw.Close()
		require.NoError(t, err)

		WithHeader(map[string]string{
			fiber.HeaderContentType: mw.FormDataContentType(),
		})(req)

		WithBody(body.Bytes())(req)
	}
}

func WithAccept(mimeType string) RequestOpt {
	return func(req *httpRequest) {
		WithHeader(map[string]string{
			fiber.HeaderAccept: mimeType,
		})(req)
	}
}

func HTMXRequest() RequestOpt {
	return func(req *httpRequest) {
		WithHeader(map[string]string{
			web.HeaderHTMXRequest: "true",
		})(req)
	}
}

func WithJSONBody(t *testing.T, v interface{}) RequestOpt {
	raw := ToJSON(t, v)

	return func(req *httpRequest) {
		req.header[fiber.HeaderContentType] = fiber.MIMEApplicationJSON

		req.body = raw
	}
}

func WithHeader(header map[string]string) RequestOpt {
	return func(req *httpRequest) {
		for k, v := range header {
			req.header[k] = v
		}
	}
}

func WithCookie(name, value string) RequestOpt {
	return func(req *httpRequest) {
		if name == "" {
			return
		}

		req.cookies = append(req.cookies, &http.Cookie{Name: name, Value: value})
	}
}

func WithSessionCookie(value string) RequestOpt {
	return func(req *httpRequest) {
		req.cookies = append(req.cookies, &http.Cookie{Name: "SESSION", Value: value})
	}
}

func WithEncryptedCookie(t *testing.T, name, value string) RequestOpt {
	return func(req *httpRequest) {
		v, err := encryptcookie.EncryptCookie(value, CookieEncryptionKey)
		require.NoError(t, err)

		WithCookie(name, v)(req)
	}
}

func Base64Encoded(t *testing.T, value any) string {
	rawValue, err := json.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}

	return base64.URLEncoding.EncodeToString(rawValue)
}

func Close(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp == nil {
		return
	}

	err := resp.Body.Close()
	if err != nil {
		t.Logf("failed to close response body %v", err)

		return
	}
}

func AssertContainsPartialHTML(t *testing.T, val string) {
	t.Helper()

	require.NotContains(t, val, "<html")
	require.NotContains(t, val, "<body")
}

func AssertContainsFullHTML(t *testing.T, val string) {
	t.Helper()

	require.Contains(t, val, "<html")
	require.Contains(t, val, "<body")
}

func AssertContainsProfile(t *testing.T, val string) {
	t.Helper()

	require.Contains(t, val, "data-testid=\"user-profile-btn\"")
}

func AssertContainsLogin(t *testing.T, val string) {
	t.Helper()

	require.Contains(t, val, "data-testid=\"user-login-btn\"")
}
