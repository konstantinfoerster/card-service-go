package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

var ErrNoClaimsInContext = fmt.Errorf("no claims in context")

const ClaimsContextKey = "claims"

func ClaimsFromCtx(ctx *fiber.Ctx) (*Claims, error) {
	u, ok := ctx.Locals(ClaimsContextKey).(*Claims)
	if ok && u != nil {
		return u, nil
	}

	return nil, ErrNoClaimsInContext
}

type MiddlewareConfig struct {
	// Extractor defines how the token claims are extracted from the request
	Extractor func(*fiber.Ctx, string) (*Claims, error)
	// Authorized runs after valid claims are found
	Authorized func(*fiber.Ctx, *Claims)
	// Key name of the session cookie
	Key string
	// AllowEmptyCookie allows unauthenticated access if true
	AllowEmptyCookie bool
}

func NewOAuthMiddleware(svc Service, opts ...func(*MiddlewareConfig)) fiber.Handler {
	c := MiddlewareConfig{
		Extractor: func(c *fiber.Ctx, cookie string) (*Claims, error) {
			if cookie == "" {
				return nil, aerrors.NewAuthorizationError(fmt.Errorf("no running session found"), "no-session")
			}

			jwtToken, err := DecodeBase64[JWT](cookie)
			if err != nil {
				return nil, err
			}

			claims, err := svc.AuthInfo(c.Context(), jwtToken.Provider, jwtToken)
			if err != nil {
				return nil, err
			}

			return claims, nil
		},
		Authorized: func(ctx *fiber.Ctx, claims *Claims) {
			ctx.Locals(ClaimsContextKey, claims)
		},
	}

	for _, optFn := range opts {
		optFn(&c)
	}

	return newTokenExtractHandler(c)
}

func WithConfig(cfg config.Oidc) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.Key = cfg.SessionCookieName
	}
}

func WithAuthorized(fn func(*fiber.Ctx, *Claims)) func(*MiddlewareConfig) {
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

			return aerrors.NewAuthorizationError(fmt.Errorf("unauthorized"), "unauthorized")
		}

		value, err := cfg.Extractor(c, cookieValue)

		if err == nil && value != nil {
			cfg.Authorized(c, value)

			return c.Next()
		}

		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		return aerrors.NewAuthorizationError(fmt.Errorf("invalid or expired session"), "unauthorized")
	}
}
