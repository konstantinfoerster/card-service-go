package oidc

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

const userContextKey = "userid"

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

func NewOauthMiddleware(cfg config.Oidc) *Middleware {
	c := Config{
		Key: cfg.SessionCookieName,
		Extractor: func(ctx *fiber.Ctx, cookie string) (interface{}, error) {
			claims, err := ExtractClaimsFromCookie(cookie)
			if err != nil {
				return nil, err
			}

			return claims.ID, nil
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
			c.Locals(userContextKey, value)

			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired session")
	}
}

type User struct {
	ID string
}

var (
	ErrNoUserInContext = common.NewAuthorizationError(fmt.Errorf("no user in context"), "no-user-found")
)

func UserFromCtx(ctx *fiber.Ctx) (User, error) {
	u, ok := ctx.Locals(userContextKey).(User)
	if ok {
		return u, nil
	}

	return User{}, ErrNoUserInContext
}
