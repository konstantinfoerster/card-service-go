package auth_test

import (
	"encoding/gob"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthMiddleware(t *testing.T) {
	gob.Register(auth.User{})
	cfg := session.Config{
		KeyLookup:  "cookie:SESSION",
		Storage:    memory.New(),
		Expiration: 24 * time.Hour,
	}
	// logged in user
	loggedInUser := auth.NewUser("test-1")
	err := cfg.Storage.Set("validSessionID", test.AsSessionData(t, auth.UserContextKey, loggedInUser), cfg.Expiration)
	require.NoError(t, err)

	// logged in but expired user
	expiredUser := auth.NewUser("expired-1")
	err = cfg.Storage.Set("expiredSessionID", test.AsSessionData(t, auth.UserContextKey, expiredUser), -24*time.Hour)
	require.NoError(t, err)

	store := session.New(cfg)

	cases := []struct {
		name           string
		cookie         *http.Cookie
		expected       *auth.User
		expectedStatus int
	}{
		{
			name: "logged in user",
			cookie: &http.Cookie{
				Name:  "SESSION",
				Value: "validSessionID",
			},
			expected:       &loggedInUser,
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired session",
			cookie: &http.Cookie{
				Name:  "SESSION",
				Value: "expiredSessionID",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "unknown session id",
			cookie: &http.Cookie{
				Name:  "SESSION",
				Value: "unknown",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "expired cookie",
			cookie: &http.Cookie{
				Name:    "SESSION",
				Value:   "dummySession",
				Expires: time.Now().Add(-48 * time.Hour),
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "no cookie",
			cookie:         &http.Cookie{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "empty cookie value",
			cookie: &http.Cookie{
				Name:  "SESSION",
				Value: "",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	t.Run("strict mode", func(t *testing.T) {
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

				app.Use(auth.NewMiddleware(store))
				app.Get("/test", func(c *fiber.Ctx) error {
					if tc.expected != nil {
						actual, err := auth.UserFromCtx(c)

						require.NoError(t, err)
						assert.Equal(t, tc.expected, &actual)

						return nil
					}

					t.Fatalf("that should have never be called without authentication")

					return nil
				})

				req := test.NewRequest(
					test.WithMethod(http.MethodGet),
					test.WithURL("/test"),
					test.WithSession(tc.cookie.Value),
				)

				resp, err := app.Test(req)

				require.NoError(t, err)
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			})
		}
	})

	t.Run("relaxed mode", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				app := fiber.New()
				store := session.New(cfg)
				app.Use(auth.NewMiddleware(store, auth.AllowUnauthorized()))
				app.Get("/test", func(c *fiber.Ctx) error {
					if tc.expected != nil {
						actual, err := auth.UserFromCtx(c)

						require.NoError(t, err)
						assert.Equal(t, tc.expected, &actual)

						return nil
					}

					actual := c.Locals(auth.UserContextKey)
					assert.Nil(t, actual)

					return nil
				})

				req := test.NewRequest(
					test.WithMethod(http.MethodGet),
					test.WithURL("/test"),
					test.WithSession(tc.cookie.Value),
				)

				resp, err := app.Test(req)

				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
	})
}
