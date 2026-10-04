package web

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var ErrUnknownAsset = errors.New("unknown asset")

// Assets maps asset URLs to fingerprinted URLs that contain a hash of the file content.
type Assets struct {
	prefix string
	// hashed contains e.g /public/css/main.css -> /public/css/main.3f2a1c9e.css
	hashed map[string]string
	// original contains e.g /public/css/main.3f2a1c9e.css -> /public/css/main.css
	original map[string]string
}

// NewAssets hashes every file in fsys once, pre-compressed variants (.br, .gz) are skipped.
func NewAssets(fsys fs.FS, prefix string) (*Assets, error) {
	a := &Assets{
		prefix:   prefix,
		hashed:   make(map[string]string),
		original: make(map[string]string),
	}

	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		ext := path.Ext(name)
		if ext == ".br" || ext == ".gz" {
			return nil
		}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}

		sum := sha256.Sum256(content)
		fingerprintLength := 8
		fingerprint := hex.EncodeToString(sum[:])[:fingerprintLength]
		hashedName := strings.TrimSuffix(name, ext) + "." + fingerprint + ext

		url := prefix + "/" + name
		hashedURL := prefix + "/" + hashedName
		a.hashed[url] = hashedURL
		a.original[hashedURL] = url

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fingerprint assets: %w", err)
	}

	return a, nil
}

// Fingerprinted returns the fingerprinted URL for a file name such as css/main.css, or an error for unknown files.
func (a *Assets) Fingerprinted(name string) (string, error) {
	hashedURL, ok := a.hashed[a.prefix+"/"+name]
	if !ok {
		return "", fmt.Errorf("asset %q: %w", name, ErrUnknownAsset)
	}

	return hashedURL, nil
}

// Original resolves a fingerprinted URL back to the URL of the real file.
func (a *Assets) Original(url string) (string, bool) {
	u, ok := a.original[url]

	return u, ok
}

// NewFingerprintMiddleware maps fingerprinted asset URLs to the real files and marks them as cacheable forever.
func NewFingerprintMiddleware(assets *Assets) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// static files are only served on GET and HEAD requests
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.Next()
		}

		original, ok := assets.Original(c.Path())
		if !ok {
			return c.Next()
		}

		// the URL changes with the content, so it never needs revalidation
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Path(original)

		return c.Next()
	}
}
