package web

import (
	"io/fs"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// NewStaticFilesMiddleware serves files from the given FS, using pre-compressed
// .br or .gz variants when the client supports them.
func NewStaticFilesMiddleware(f fs.FS, prefix string) fiber.Handler {
	staticFs := &fasthttp.FS{
		FS:             f,
		Compress:       true,
		CompressBrotli: true,
		// maps encoding to matching file endings
		CompressedFileSuffixes: map[string]string{
			"br":   ".br",
			"gzip": ".gz",
		},
		// strip the route prefix, paths in f are relative to it,
		// e.g. /public/css/main.css -> /css/main.css
		PathRewrite: fasthttp.NewPathPrefixStripper(len(prefix)),
	}
	handler := staticFs.NewRequestHandler()

	return func(c *fiber.Ctx) error {
		// serve static files only on GET and HEAD requests
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.Next()
		}

		// prevents caches from serving a compressed response to clients that don't support it
		c.Vary(fiber.HeaderAcceptEncoding)
		handler(c.Context())

		return nil
	}
}
