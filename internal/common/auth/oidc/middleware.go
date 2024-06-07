package oidc

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
)

type MiddlewareConfig struct {
	Extractor        func(*fiber.Ctx, string) (*auth.User, error)
	Key              string
	AllowEmptyCookie bool
}

func NewOauthMiddleware(svc UserService, options ...func(*MiddlewareConfig)) fiber.Handler {
	c := MiddlewareConfig{
		Extractor: func(ctx *fiber.Ctx, cookie string) (*auth.User, error) {
			if cookie == "" {
				return nil, common.NewAuthorizationError(fmt.Errorf("no running session found"), "no-session")
			}

			jwtToken, err := commonhttp.DecodeBase64[JSONWebToken](cookie)
			if err != nil {
				return nil, err
			}

			return svc.GetAuthenticatedUser(jwtToken.Provider, jwtToken)
		},
	}

	for _, optionFn := range options {
		optionFn(&c)
	}

	return newTokenExtractHandler(c)
}

func FromConfig(cfg config.Oidc) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.Key = cfg.SessionCookieName
	}
}

func AllowEmptyCookie() func(*MiddlewareConfig) {
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

			return common.NewAuthorizationError(fmt.Errorf("unauthorized"), "unauthorized")
		}

		value, err := cfg.Extractor(c, cookieValue)

		if err == nil && value != nil {
			auth.UserToCtx(c, value)

			return c.Next()
		}

		if err != nil {
			return common.NewAuthorizationError(err, "unauthorized")
		}

		return common.NewAuthorizationError(fmt.Errorf("invalid or expired session"), "unauthorized")
	}
}
