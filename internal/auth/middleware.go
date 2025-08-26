package auth

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

var (
	ErrNoUserInContext = errors.New("no user in context")
	ErrInvalidate      = errors.New("invalidation failed")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidSession  = errors.New("invalid or expired session")
)

const UserContextKey = "session_user"

// UserFromCtx returns an authenticated User or an ErrNoUserInContext if there is no user.
func UserFromCtx(ctx *fiber.Ctx) (User, error) {
	u, ok := ctx.Locals(UserContextKey).(User)
	if ok && u.Valid() {
		return u, nil
	}

	return User{}, ErrNoUserInContext
}

type MiddlewareConfig struct {
	// extractor defines how the session is extracted from the request
	extractor func(*fiber.Ctx) (User, error)
	// AllowEmptyCookie allows unauthenticated access if true
	AllowEmptyCookie bool
}

type MiddlewareOpt func(*MiddlewareConfig)

func NewMiddleware(store *session.Store, opts ...MiddlewareOpt) fiber.Handler {
	c := MiddlewareConfig{
		extractor: func(c *fiber.Ctx) (User, error) {
			sess, err := store.Get(c)
			if err != nil {
				slog.Error("failed to get session from store", slog.Any("error", err))

				return User{}, errors.Join(ErrInvalidSession, err)
			}

			if sess.Fresh() {
				return User{}, ErrInvalidSession
			}

			user, ok := sess.Get(UserContextKey).(User)
			if !ok {
				return User{}, ErrNoUserInContext
			}

			if !user.Valid() {
				return User{}, ErrInvalidSession
			}

			return user, nil
		},
	}

	for _, optFn := range opts {
		if optFn == nil {
			continue
		}

		optFn(&c)
	}

	return newExtractHandler(c)
}

func AllowUnauthorized() MiddlewareOpt {
	return func(c *MiddlewareConfig) {
		c.AllowEmptyCookie = true
	}
}

func newExtractHandler(config ...MiddlewareConfig) fiber.Handler {
	var cfg MiddlewareConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.extractor == nil {
		panic("auth middleware requires an extractor function")
	}

	return func(c *fiber.Ctx) error {
		user, err := cfg.extractor(c)
		if err != nil {
			if cfg.AllowEmptyCookie {
				return c.Next()
			}

			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		c.Locals(UserContextKey, user)

		return c.Next()
	}
}
