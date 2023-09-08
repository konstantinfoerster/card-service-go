package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewHttpServerErrorHandler(t *testing.T) {
	cfg := &config.Server{
		Cookie: config.Cookie{
			EncryptionKey: "01234567890123456789012345678901",
		},
	}
	cases := []struct {
		name       string
		appErr     common.AppError
		statusCode int
	}{
		{
			name:       "Invalid input",
			appErr:     common.NewInvalidInputError(fmt.Errorf("some error"), "mykey", "mymsg"),
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Unauthorized",
			appErr:     common.NewAuthorizationError(fmt.Errorf("some error"), "myKey"),
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "Unknown error",
			appErr:     common.NewUnknownError(fmt.Errorf("some error"), "myKey"),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := server.NewHTTPServer(cfg)
			srv.RegisterAPIRoutes(func(app fiber.Router) {
				app.Get("/", func(c *fiber.Ctx) error {
					return tc.appErr
				})
			})
			req := httptest.NewRequest(http.MethodGet, "https://localhost/", nil)

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			assert.NoError(t, err)
			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Contains(t, resp.Header.Get(fiber.HeaderContentType), "application/json")
			result := commontest.FromJSON[problemjson.ProblemJSON](t, resp)
			assert.Equal(t, tc.statusCode, result.Status)
			assert.NotEmpty(t, result.Key)
		})
	}
}

func TestNewHttpServerCookieEncryption(t *testing.T) {
	cfg := &config.Server{
		Cookie: config.Cookie{
			EncryptionKey: "01234567890123456789012345678901",
		},
	}
	srv := server.NewHTTPServer(cfg)
	srv.RegisterAPIRoutes(func(app fiber.Router) {
		app.Get("/", func(c *fiber.Ctx) error {
			c.Cookie(&fiber.Cookie{
				Name:  "TEST",
				Value: "myValue",
			})

			return nil
		})
	})
	req := httptest.NewRequest(http.MethodGet, "https://localhost/", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.NotEqual(t, "myValue", resp.Cookies()[0].Value)
}
