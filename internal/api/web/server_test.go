package web_test

import (
	"context"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewServerErrorHandler(t *testing.T) {
	cases := []struct {
		name       string
		appErr     aerrors.AppError
		statusCode int
	}{
		{
			name:       "Invalid input",
			appErr:     aerrors.NewInvalidInputError(assert.AnError, "mykey", "mymsg"),
			statusCode: web.StatusBadRequest,
		},
		{
			name:       "Unauthorized",
			appErr:     aerrors.NewAuthorizationError(assert.AnError, "myKey"),
			statusCode: web.StatusUnauthorized,
		},
		{
			name:       "Unknown error",
			appErr:     aerrors.NewUnknownError(assert.AnError, "myKey"),
			statusCode: web.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := web.NewTestServer()
			srv.RegisterRoutes(func(app fiber.Router) {
				app.Get("/", func(c *fiber.Ctx) error {
					return tc.appErr
				})
			})
			req := httptest.NewRequest(web.MethodGet, "https://localhost/", nil)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, web.ContentType, resp.Header.Get(fiber.HeaderContentType))
			result := test.FromJSON[web.ProblemJSON](t, resp.Body)
			assert.Equal(t, tc.statusCode, result.Status)
			assert.NotEmpty(t, result.Key)
		})
	}
}

func TestNewServerCookieEncryption(t *testing.T) {
	srv := web.NewTestServer()
	srv.RegisterRoutes(func(app fiber.Router) {
		app.Get("/", func(c *fiber.Ctx) error {
			c.Cookie(&fiber.Cookie{
				Name:  "TEST",
				Value: "myValue",
			})

			return nil
		})
	})
	req := httptest.NewRequest(web.MethodGet, "https://localhost/", nil)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.NotEqual(t, "myValue", resp.Cookies()[0].Value)
}

func TestNewServerShutdownOnInteruptSignal(t *testing.T) {
	srv := web.NewTestServer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return srv.Run(ctx)
	})
	waitForStartup(t, srv)

	killErr := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	require.NoError(t, killErr)
	err := errg.Wait()

	require.NoError(t, err)
	assert.False(t, srv.IsRunning())
}

func waitForStartup(t *testing.T, srv *web.Server) {
	maxTries := 3
	count := 0
	for {
		time.Sleep(10 * time.Millisecond)
		if srv.IsRunning() {
			break
		}

		count++
		if count == maxTries {
			assert.Fail(t, "server not start in time")

			break
		}
	}
}
