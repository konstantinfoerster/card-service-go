package web_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fingerprint(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])[:8]
}

func TestAssetsFingerprinted(t *testing.T) {
	fsys := fstest.MapFS{
		"js/app.js": {Data: []byte("original")},
	}
	assets, err := web.NewAssets(fsys, "/public")
	require.NoError(t, err)

	url, err := assets.Fingerprinted("js/app.js")

	require.NoError(t, err)
	assert.Equal(t, "/public/js/app."+fingerprint("original")+".js", url)
}

func TestAssetsFingerprinted_Unknown(t *testing.T) {
	fsys := fstest.MapFS{
		"js/app.js":    {Data: []byte("original")},
		"js/app.js.br": {Data: []byte("brotli")},
		"js/app.js.gz": {Data: []byte("gzip")},
	}
	assets, err := web.NewAssets(fsys, "/public")
	require.NoError(t, err)

	for _, name := range []string{"js/missing.js", "js/app.js.br", "js/app.js.gz"} {
		t.Run(name, func(t *testing.T) {
			_, err := assets.Fingerprinted(name)

			require.Error(t, err)
		})
	}
}

func TestAssetsOriginal(t *testing.T) {
	fsys := fstest.MapFS{
		"js/app.js": {Data: []byte("original")},
	}
	assets, err := web.NewAssets(fsys, "/public")
	require.NoError(t, err)

	original, ok := assets.Original("/public/js/app." + fingerprint("original") + ".js")

	assert.True(t, ok)
	assert.Equal(t, "/public/js/app.js", original)
}

func TestAssetsOriginal_Unknown(t *testing.T) {
	fsys := fstest.MapFS{
		"js/app.js": {Data: []byte("original")},
	}
	assets, err := web.NewAssets(fsys, "/public")
	require.NoError(t, err)

	for _, url := range []string{"/public/js/app.js", "/public/js/app.00000000.js", "/public/js/missing.js"} {
		t.Run(url, func(t *testing.T) {
			_, ok := assets.Original(url)

			assert.False(t, ok)
		})
	}
}

func TestNewFingerprintMiddleware(t *testing.T) {
	type response struct {
		Path         string
		CacheControl string
	}

	fsys := fstest.MapFS{
		"js/app.js": {Data: []byte("original")},
	}
	assets, err := web.NewAssets(fsys, "/public")
	require.NoError(t, err)

	cases := []struct {
		name   string
		method string
		path   string
		want   response
	}{
		{
			name:   "fingerprinted",
			method: fiber.MethodGet,
			path:   "/public/js/app." + fingerprint("original") + ".js",
			want: response{
				Path:         "/public/js/app.js",
				CacheControl: "public, max-age=31536000, immutable",
			},
		},
		{
			name:   "plain",
			method: fiber.MethodGet,
			path:   "/public/js/app.js",
			want:   response{Path: "/public/js/app.js"},
		},
		{
			name:   "stale fingerprint",
			method: fiber.MethodGet,
			path:   "/public/js/app.00000000.js",
			want:   response{Path: "/public/js/app.00000000.js"},
		},
		{
			name:   "fingerprinted head",
			method: fiber.MethodHead,
			path:   "/public/js/app." + fingerprint("original") + ".js",
			want: response{
				Path:         "/public/js/app.js",
				CacheControl: "public, max-age=31536000, immutable",
			},
		},
		{
			name:   "fingerprinted post passes through",
			method: fiber.MethodPost,
			path:   "/public/js/app." + fingerprint("original") + ".js",
			want:   response{Path: "/public/js/app." + fingerprint("original") + ".js"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use("/public", web.NewFingerprintMiddleware(assets), func(c *fiber.Ctx) error {
				// echo the path as header, HEAD responses have no body
				c.Set("X-Path", c.Path())

				return nil
			})
			req := test.NewRequest(
				t.Context(),
				test.WithMethod(tc.method),
				test.WithURL(tc.path),
			)

			resp, err := app.Test(req, -1)
			defer test.Close(t, resp)
			require.NoError(t, err)

			assert.Equal(t, tc.want, response{
				Path:         resp.Header.Get("X-Path"),
				CacheControl: resp.Header.Get(fiber.HeaderCacheControl),
			})
		})
	}
}
