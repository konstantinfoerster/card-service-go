package web_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := web.Auth{
		HeaderUserID:    "X-Request-User",
		HeaderUserEmail: "X-Request-Email",
	}
	cases := []struct {
		name           string
		header         map[string]string
		expected       web.User
		expectedStatus int
	}{
		{
			name: "logged in user",
			header: map[string]string{
				"X-Request-User":  "myuser",
				"X-Request-Email": "myuser@localhost",
			},
			expected:       web.NewUser("myuser").WithEmail("myuser@localhost"),
			expectedStatus: http.StatusOK,
		},
		{
			name: "logged in user without email",
			header: map[string]string{
				"X-Request-User": "myuser",
			},
			expected:       web.NewUser("myuser"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not logged in",
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

				app.Use(web.NewMiddleware(cfg))
				app.Get("/test", func(c *fiber.Ctx) error {
					if tc.expected.ID != "" {
						actual, err := web.UserFromCtx(c)

						require.NoError(t, err)
						assert.Equal(t, tc.expected, actual)

						return nil
					}

					t.Fatalf("that should have never be called without authentication")

					return nil
				})

				req := test.NewRequest(
					t.Context(),
					test.WithMethod(http.MethodGet),
					test.WithURL("/test"),
					test.WithHeader(tc.header),
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
				app.Use(web.NewMiddleware(cfg, web.AllowUnauthorized()))
				app.Get("/test", func(c *fiber.Ctx) error {
					if tc.expected.ID != "" {
						actual, err := web.UserFromCtx(c)

						require.NoError(t, err)
						assert.Equal(t, tc.expected, actual)

						return nil
					}

					actual := c.Locals(web.UserContextKey)
					assert.Nil(t, actual)

					return nil
				})

				req := test.NewRequest(
					t.Context(),
					test.WithMethod(http.MethodGet),
					test.WithURL("/test"),
					test.WithHeader(tc.header),
				)

				resp, err := app.Test(req)

				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
	})
}

func TestUserFromCtx(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(web.UserContextKey, web.NewUser("myuser"))

		user, err := web.UserFromCtx(c)

		require.NoError(t, err)
		assert.NotNil(t, user)

		return nil
	})
	req := test.NewRequest(
		t.Context(),
		test.WithMethod(http.MethodGet),
		test.WithURL("/test"),
	)

	_, err := app.Test(req)

	require.NoError(t, err)
}

func TestUserFromCtx_InvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		setUser func(c *fiber.Ctx)
	}{
		{
			name: "nil user",
			setUser: func(c *fiber.Ctx) {
				c.Locals(web.UserContextKey, nil)
			},
		},
		{
			name: "no key",
			setUser: func(c *fiber.Ctx) {
			},
		},
		{
			name: "wrong type",
			setUser: func(c *fiber.Ctx) {
				c.Locals(web.UserContextKey, "wrongType")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				tc.setUser(c)

				user, err := web.UserFromCtx(c)

				assert.Equal(t, web.User{}, user)
				require.ErrorIs(t, err, web.ErrNoUserInContext)

				return nil
			})
			req := test.NewRequest(
				t.Context(),
				test.WithMethod(http.MethodGet),
				test.WithURL("/test"),
			)

			_, err := app.Test(req)

			require.NoError(t, err)
		})
	}
}
