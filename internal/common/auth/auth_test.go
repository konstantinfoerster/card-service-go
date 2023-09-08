package auth_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserFromCtx(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(auth.UserContextKey, &auth.User{})

		user, err := auth.UserFromCtx(c)

		assert.NoError(t, err, auth.ErrNoUserInContext)
		assert.NotNil(t, user)

		return nil
	})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("/test"),
	)

	_, err := app.Test(req)

	require.NoError(t, err)
}

func TestUserFromCtxInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		setUser func(c *fiber.Ctx)
	}{
		{
			name: "nil user",
			setUser: func(c *fiber.Ctx) {
				c.Locals(auth.UserContextKey, nil)
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
				c.Locals(auth.UserContextKey, "wrongType")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				tc.setUser(c)

				user, err := auth.UserFromCtx(c)

				assert.Nil(t, user)
				assert.ErrorIs(t, err, auth.ErrNoUserInContext)

				return nil
			})
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("/test"),
			)

			_, err := app.Test(req)

			require.NoError(t, err)
		})
	}
}
