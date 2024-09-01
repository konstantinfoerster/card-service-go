package web_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserFromCtx(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(web.UserContextKey, &web.User{})

		user, err := web.UserFromCtx(c)

		require.NoError(t, err)
		assert.NotNil(t, user)

		return nil
	})
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("/test"),
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

				assert.Nil(t, user)
				require.ErrorIs(t, err, web.ErrNoUserInContext)

				return nil
			})
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("/test"),
			)

			_, err := app.Test(req)

			require.NoError(t, err)
		})
	}
}
