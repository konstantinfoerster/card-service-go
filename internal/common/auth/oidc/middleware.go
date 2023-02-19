package oidc

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type Config struct {
	Key       string
	Extractor func(*fiber.Ctx, string) (interface{}, error)
}

type Middleware struct {
	Handler fiber.Handler
}

func (m *Middleware) Middleware() fiber.Handler {
	return m.Handler
}

func NewOauthMiddleware(cfg config.Oidc, sp *SupportedProvider) *Middleware {
	c := Config{
		Key: cfg.SessionCookieName,
		Extractor: func(ctx *fiber.Ctx, cookie string) (interface{}, error) {
			claims, err := ExtractClaimsFromCookie(cookie, sp)
			if err != nil {
				return nil, err
			}

			return ClaimsToUser(claims), nil
		},
	}

	return &Middleware{Handler: newTokenExtractHandler(c)}
}

func newTokenExtractHandler(config ...Config) fiber.Handler {
	// Init config
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Key == "" {
		cfg.Key = "SESSION"
	}
	if cfg.Extractor == nil {
		panic("OAuth handler requires an extractor function")
	}

	return func(c *fiber.Ctx) error {
		// Extract and verify key
		key := c.Cookies(cfg.Key)
		if key == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		value, err := cfg.Extractor(c, key)

		if err == nil && value != nil {
			c.Locals(auth.UserContextKey, value)

			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired session")
	}
}
