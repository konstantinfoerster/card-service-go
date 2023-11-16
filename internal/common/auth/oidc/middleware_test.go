package oidc_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserService struct {
	FakeGetAuthenticatedUser func() (*auth.User, error)
}

var _ oidc.UserService = (*mockUserService)(nil)

func (s *mockUserService) GetAuthenticatedUser(_ string, _ *oidc.JSONWebToken) (*auth.User, error) {
	if s.FakeGetAuthenticatedUser != nil {
		return s.FakeGetAuthenticatedUser()
	}

	return nil, fmt.Errorf("unexpected function call, no fake implementation provided")
}

func TestOauthMiddleware(t *testing.T) {
	expectedUser := &auth.User{
		ID:       "1",
		Username: "test@localhost",
	}
	svc := &mockUserService{
		FakeGetAuthenticatedUser: func() (*auth.User, error) {
			return expectedUser, nil
		},
	}
	app := fiber.New()
	app.Use(oidc.NewOauthMiddleware(svc))
	app.Get("/test", func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)

		return c.SendString("OK")
	})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("/test"),
		commontest.WithCookie(&http.Cookie{
			Name:  "SESSION",
			Value: commontest.Base64Encoded(t, &oidc.JSONWebToken{}),
		}),
	)

	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOauthMiddlewareUseConfiguredCookie(t *testing.T) {
	expectedUser := &auth.User{}
	svc := &mockUserService{
		FakeGetAuthenticatedUser: func() (*auth.User, error) {
			return expectedUser, nil
		},
	}
	cfg := config.Oidc{SessionCookieName: "MY_SESSION"}
	app := fiber.New()
	app.Use(oidc.NewOauthMiddleware(svc, oidc.FromConfig(cfg)))
	app.Get("/test", func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)

		return c.SendString("OK")
	})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("/test"),
		commontest.WithCookie(&http.Cookie{
			Name:  cfg.SessionCookieName,
			Value: commontest.Base64Encoded(t, &oidc.JSONWebToken{}),
		}),
	)

	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOauthMiddlewareError(t *testing.T) {
	svc := &mockUserService{
		FakeGetAuthenticatedUser: func() (*auth.User, error) {
			return &auth.User{}, nil
		},
	}
	cfg := config.Oidc{SessionCookieName: "SESSION"}
	cases := []struct {
		name   string
		cookie *http.Cookie
		svc    oidc.UserService
	}{
		{
			name: "service error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: commontest.Base64Encoded(t, &oidc.JSONWebToken{}),
			},
			svc: &mockUserService{
				FakeGetAuthenticatedUser: func() (*auth.User, error) {
					return nil, fmt.Errorf("some error")
				},
			},
		},
		{
			name:   "no cookie",
			cookie: nil,
			svc:    svc,
		},
		{
			name: "empty cookie value",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: "",
			},
			svc: svc,
		},
		{
			name: "decode base64 cookie error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: ";a",
			},
			svc: svc,
		},
		{
			name: "decode json cookie error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: base64.URLEncoding.EncodeToString([]byte("{xyz\": 1}")),
			},
			svc: svc,
		},
		{
			name: "nil user from service without error",
			cookie: &http.Cookie{
				Name:  cfg.SessionCookieName,
				Value: commontest.Base64Encoded(t, &oidc.JSONWebToken{}),
			},
			svc: &mockUserService{
				FakeGetAuthenticatedUser: func() (*auth.User, error) {
					return nil, fmt.Errorf("that should never be called")
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: problemjson.RespondWithProblemJSON,
			})
			app.Use(oidc.NewOauthMiddleware(tc.svc, oidc.FromConfig(cfg)))
			app.Get("/test", func(c *fiber.Ctx) error {
				t.Fatalf("that should never be called")

				return nil
			})

			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("/test"),
				commontest.WithCookie(tc.cookie),
			)

			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}
