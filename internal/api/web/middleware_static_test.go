package web_test

import (
	"io"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStaticFilesMiddleware(t *testing.T) {
	type response struct {
		Status      int
		Encoding    string
		ContentType string
		Vary        string
		Body        string
	}

	fsys := fstest.MapFS{
		"js/app.js":    {Data: []byte("original")},
		"js/app.js.br": {Data: []byte("brotli")},
		"js/app.js.gz": {Data: []byte("gzip")},
	}

	cases := []struct {
		name           string
		method         string
		path           string
		acceptEncoding string
		want           response
	}{
		{
			name:           "brotli",
			method:         fiber.MethodGet,
			path:           "/public/js/app.js",
			acceptEncoding: "br",
			want: response{
				Status:      web.StatusOK,
				Encoding:    "br",
				ContentType: fiber.MIMETextJavaScriptCharsetUTF8,
				Vary:        fiber.HeaderAcceptEncoding,
				Body:        "brotli",
			},
		},
		{
			name:           "gzip",
			method:         fiber.MethodGet,
			path:           "/public/js/app.js",
			acceptEncoding: "gzip",
			want: response{
				Status:      web.StatusOK,
				Encoding:    "gzip",
				ContentType: fiber.MIMETextJavaScriptCharsetUTF8,
				Vary:        fiber.HeaderAcceptEncoding,
				Body:        "gzip",
			},
		},
		{
			name:           "brotli preferred",
			method:         fiber.MethodGet,
			path:           "/public/js/app.js",
			acceptEncoding: "gzip, deflate, br",
			want: response{
				Status:      web.StatusOK,
				Encoding:    "br",
				ContentType: fiber.MIMETextJavaScriptCharsetUTF8,
				Vary:        fiber.HeaderAcceptEncoding,
				Body:        "brotli",
			},
		},
		{
			name:           "no encoding",
			method:         fiber.MethodGet,
			path:           "/public/js/app.js",
			acceptEncoding: "",
			want: response{
				Status:      web.StatusOK,
				ContentType: fiber.MIMETextJavaScriptCharsetUTF8,
				Vary:        fiber.HeaderAcceptEncoding,
				Body:        "original",
			},
		},
		{
			name:           "not found",
			method:         fiber.MethodGet,
			path:           "/public/js/missing.js",
			acceptEncoding: "br",
			want: response{
				Status:      web.StatusNotFound,
				ContentType: fiber.MIMETextPlainCharsetUTF8,
				Body:        "Cannot open requested path",
			},
		},
		{
			name:           "head",
			method:         fiber.MethodHead,
			path:           "/public/js/app.js",
			acceptEncoding: "br",
			want: response{
				Status:      web.StatusOK,
				Encoding:    "br",
				ContentType: fiber.MIMETextJavaScriptCharsetUTF8,
				Vary:        fiber.HeaderAcceptEncoding,
			},
		},
		{
			name:           "other method passes through",
			method:         fiber.MethodPost,
			path:           "/public/js/app.js",
			acceptEncoding: "br",
			want: response{
				Status:      web.StatusNotFound,
				ContentType: fiber.MIMETextPlainCharsetUTF8,
				Body:        "Cannot POST /public/js/app.js",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use("/public", web.NewStaticFilesMiddleware(fsys, "/public"))
			req := test.NewRequest(
				t.Context(),
				test.WithMethod(tc.method),
				test.WithURL(tc.path),
				test.WithHeader(map[string]string{
					fiber.HeaderAcceptEncoding: tc.acceptEncoding,
				}),
			)

			resp, err := app.Test(req, -1)
			defer test.Close(t, resp)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.want, response{
				Status:      resp.StatusCode,
				Encoding:    resp.Header.Get(fiber.HeaderContentEncoding),
				ContentType: resp.Header.Get(fiber.HeaderContentType),
				Vary:        resp.Header.Get(fiber.HeaderVary),
				Body:        string(body),
			})
		})
	}
}
