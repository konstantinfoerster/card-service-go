package web

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

var (
	ErrNoUserInContext = errors.New("no user in context")
	ErrInvalidate      = errors.New("invalidation failed")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidSession  = errors.New("invalid or expired session")
)

const UserContextKey = "session_user"

// User represents an authenticated user.
type User struct {
	ID    string
	Email string
}

// NewUser creates a new User.
func NewUser(id string) User {
	return User{ID: id}
}

func (u User) WithEmail(email string) User {
	u.Email = email

	return u
}

func (u User) Valid() bool {
	return strings.TrimSpace(u.ID) != ""
}

// UserFromCtx returns an authenticated User or an ErrNoUserInContext if there is no user.
func UserFromCtx(ctx *fiber.Ctx) (User, error) {
	u, ok := ctx.Locals(UserContextKey).(User)
	if ok && u.Valid() {
		return u, nil
	}

	return User{}, ErrNoUserInContext
}

type MiddlewareConfig struct {
	// extractor defines how the user data is extracted from the request
	extractor func(*fiber.Ctx) (User, error)
	// AllowUnauthorized allows unauthenticated access if true
	AllowUnauthorized bool
}

type MiddlewareOpt func(*MiddlewareConfig)

func NewMiddleware(cfg Auth, opts ...MiddlewareOpt) fiber.Handler {
	c := MiddlewareConfig{
		extractor: func(c *fiber.Ctx) (User, error) {
			user := NewUser(c.Get(cfg.HeaderUserID)).
				WithEmail(c.Get(cfg.HeaderUserEmail))

			if cfg.TestMode {
				user = NewUser(cfg.UserID).WithEmail(cfg.UserEmail)
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
		c.AllowUnauthorized = true
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
			if cfg.AllowUnauthorized {
				return c.Next()
			}

			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		c.Locals(UserContextKey, user)

		return c.Next()
	}
}
