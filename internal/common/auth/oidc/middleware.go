package oidc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Key       string
	Extractor func(*fiber.Ctx, string) (*auth.User, error)
}

func NewOauthMiddleware(cfg config.Oidc, svc UserService) fiber.Handler {
	c := Config{
		Key: cfg.SessionCookieName,
		Extractor: func(ctx *fiber.Ctx, cookie string) (*auth.User, error) {
			jwtToken, err := jwtFromCookie(cookie)
			if err != nil {
				return nil, err
			}

			return svc.GetAuthenticatedUser(jwtToken.Provider, jwtToken)
		},
	}

	return newTokenExtractHandler(c)
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
			return common.NewAuthorizationError(fmt.Errorf("unauthorized"), "unauthorized")
		}

		value, err := cfg.Extractor(c, key)

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

func jwtFromCookie(cookie string) (*JSONWebToken, error) {
	if cookie == "" {
		return nil, common.NewAuthorizationError(fmt.Errorf("no running session found"), "no-session")
	}

	rawJwt, err := base64.URLEncoding.DecodeString(cookie)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode base64 string")

		return nil, common.NewUnknownError(err, "unable-to-decode-token")
	}

	var jwtToken JSONWebToken
	decoder := json.NewDecoder(bytes.NewReader(rawJwt))
	err = decoder.Decode(&jwtToken)
	if err != nil {
		log.Error().Err(err).Msgf("failed to decode jwt %s", rawJwt)

		return nil, common.NewUnknownError(err, "unable-to-decode-token")
	}

	return &jwtToken, nil
}
