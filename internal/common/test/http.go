package commontest

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/stretchr/testify/require"
)

const CookieEncryptionKey = "01234567890123456789012345678901"

type httpRequest struct {
	header  map[string]string
	url     string
	method  string
	body    []byte
	cookies []*http.Cookie
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

func WithMultipartFile(t *testing.T, f io.Reader, name string) func(*httpRequest) {
	return func(req *httpRequest) {
		body := new(bytes.Buffer)
		mw := multipart.NewWriter(body)
		w, err := mw.CreateFormFile("file", name)
		require.NoError(t, err)

		_, err = io.Copy(w, f)
		require.NoError(t, err)
		err = mw.Close()
		require.NoError(t, err)

		req.header[fiber.HeaderContentType] = mw.FormDataContentType()
		req.body = body.Bytes()
	}
}

func WithAccept(mimeType string) func(*httpRequest) {
	return func(req *httpRequest) {
		req.header[fiber.HeaderAccept] = mimeType
	}
}

func HTMXRequest() func(*httpRequest) {
	return func(req *httpRequest) {
		req.header[commonhttp.HeaderHTMXRequest] = "true"
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
