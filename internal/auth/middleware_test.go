package auth_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthMiddleware(t *testing.T) {
	expectedClaims := auth.NewClaims("test-1", "test@localhost")
	provider := auth.NewFakeProvider(auth.WithClaims(expectedClaims))
	svc := auth.New(auth.Config{}, auth.NewProviders(provider))
	app := fiber.New()
	app.Use(auth.NewOAuthMiddleware(svc))
	app.Get("/test", func(c *fiber.Ctx) error {
		claims, err := auth.ClaimsFromCtx(c)

		require.NoError(t, err)
		assert.Equal(t, expectedClaims, claims)

		return c.SendString("OK")
	})
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("/test"),
		test.WithCookie("SESSION", test.Base64Encoded(t, provider.Token("test-1"))),
	)

	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOAuthMiddlewareWithCustomCookieName(t *testing.T) {
	validClaims := auth.NewClaims("test-1", "test@localhost")
	oCfg := auth.Config{SessionCookieName: "MY_SESSION"}
	provider := auth.NewFakeProvider(auth.WithClaims(validClaims))
	svc := auth.New(oCfg, auth.NewProviders(provider))
	app := fiber.New()
	app.Use(auth.NewOAuthMiddleware(svc, auth.WithConfig(oCfg)))
	app.Get("/test", func(c *fiber.Ctx) error {
		claims, err := auth.ClaimsFromCtx(c)

		require.NoError(t, err)
		assert.Equal(t, validClaims, claims)

		return c.SendString("OK")
	})
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("/test"),
		test.WithCookie("MY_SESSION", test.Base64Encoded(t, provider.Token("test-1"))),
	)

	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOAuthMiddlewareError(t *testing.T) {
	cfg := auth.Config{SessionCookieName: "SESSION"}
	provider := auth.NewFakeProvider()
	svc := auth.New(cfg, auth.NewProviders(provider))
	cases := []struct {
		name   string
		cookie *http.Cookie
	}{
		{
			name: "invalid access token",
			cookie: &http.Cookie{
				Name: cfg.SessionCookieName,
				Value: test.Base64Encoded(t, &auth.JWT{
					Provider:    provider.GetName(),
					AccessToken: "invalidToken",
				}),
			},
		},
		{
			name: "unknown provider",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: base64.URLEncoding.EncodeToString([]byte("{\"provider\": \"unknown\"}")),
			},
		},
		{
			name:   "no cookie",
			cookie: &http.Cookie{},
		},
		{
			name: "empty cookie value",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: "",
			},
		},
		{
			name: "decode base64 cookie error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: ";a",
			},
		},
		{
			name: "decode json cookie error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: base64.URLEncoding.EncodeToString([]byte("{xyz\": 1}")),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					var appErr aerrors.AppError
					if !errors.As(err, &appErr) {
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					}

					switch appErr.ErrorType {
					case aerrors.ErrAuthorization:
						return c.Status(http.StatusUnauthorized).SendString(err.Error())
					default:
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					}
				},
			})
			app.Use(auth.NewOAuthMiddleware(svc, auth.WithConfig(cfg)))
			app.Get("/test", func(c *fiber.Ctx) error {
				t.Fatalf("that should never be called")

				return nil
			})

			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("/test"),
				test.WithCookie(tc.cookie.Name, tc.cookie.Value),
			)

			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}
