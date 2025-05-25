package auth

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

var (
	ErrNoClaimsInContext = errors.New("no claims in context")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidSession    = errors.New("invalid or expired session")
)

type Service interface {
	AuthInfo(ctx context.Context, provider string, token *JWT) (Claims, error)
}

const ClaimsContextKey = "claims"

func ClaimsFromCtx(ctx *fiber.Ctx) (Claims, error) {
	u, ok := ctx.Locals(ClaimsContextKey).(Claims)
	if ok && u.ID != "" {
		return u, nil
	}

	return Claims{}, ErrNoClaimsInContext
}

type MiddlewareConfig struct {
	// Extractor defines how the token claims are extracted from the request
	Extractor func(*fiber.Ctx, string) (Claims, error)
	// Authorized runs after valid claims are found
	Authorized func(*fiber.Ctx, Claims)
	// Key name of the session cookie
	Key string
	// AllowEmptyCookie allows unauthenticated access if true
	AllowEmptyCookie bool
}

func NewOAuthMiddleware(svc Service, opts ...func(*MiddlewareConfig)) fiber.Handler {
	c := MiddlewareConfig{
		Extractor: func(c *fiber.Ctx, cookie string) (Claims, error) {
			if cookie == "" {
				return Claims{}, ErrUnauthorized
			}

			jwtToken, err := DecodeSession(cookie)
			if err != nil {
				return Claims{}, err
			}

			claims, err := svc.AuthInfo(c.Context(), jwtToken.Provider, jwtToken)
			if err != nil {
				return Claims{}, err
			}

			return claims, nil
		},
		Authorized: func(ctx *fiber.Ctx, claims Claims) {
			ctx.Locals(ClaimsContextKey, claims)
		},
	}

	for _, optFn := range opts {
		optFn(&c)
	}

	return newTokenExtractHandler(c)
}

func WithConfig(cfg Config) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.Key = cfg.SessionCookieName
	}
}

func WithAuthorized(fn func(*fiber.Ctx, Claims)) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.Authorized = fn
	}
}
func AllowUnauthorized() func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.AllowEmptyCookie = true
	}
}

func newTokenExtractHandler(config ...MiddlewareConfig) fiber.Handler {
	// Init config
	var cfg MiddlewareConfig
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
		cookieValue := c.Cookies(cfg.Key)
		if cookieValue == "" {
			if cfg.AllowEmptyCookie {
				return c.Next()
			}

			return aerrors.NewAuthorizationError(ErrUnauthorized, "unauthorized")
		}

		value, err := cfg.Extractor(c, cookieValue)
		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		if value.ID == "" {
			return aerrors.NewAuthorizationError(ErrInvalidSession, "unauthorized")
		}

		cfg.Authorized(c, value)

		return c.Next()
	}
}
