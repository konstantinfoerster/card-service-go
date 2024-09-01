package web

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

var (
	ErrNoUserInContext = fmt.Errorf("no user in context")
)

const UserContextKey = "userid"

// NewUser creates a new User.
func NewUser(id, username string) *User {
	return &User{ID: id, Username: username}
}

// User represents an authenticated user in the system.
type User struct {
	ID       string
	Username string
}

// UserFromCtx returns an authenticated User or an ErrNoUserInContext if there is no user.
func UserFromCtx(ctx *fiber.Ctx) (*User, error) {
	u, ok := ctx.Locals(UserContextKey).(*User)
	if ok && u != nil {
		return u, nil
	}

	return nil, ErrNoUserInContext
}

// NewAuthMiddleware provides fiber.Handler that can be used to ensure an authentciated access.
func NewAuthMiddleware(cfg config.Oidc, svc auth.Service) AuthMiddleware {
	authFn := func(ctx *fiber.Ctx, claims *auth.Claims) {
		u := NewUser(claims.ID, claims.Email)
		ctx.Locals(UserContextKey, u)
	}

	relaxed := auth.NewOAuthMiddleware(svc, auth.WithAuthorized(authFn), auth.WithConfig(cfg), auth.AllowUnauthorized())
	required := auth.NewOAuthMiddleware(svc, auth.WithAuthorized(authFn), auth.WithConfig(cfg))

	return AuthMiddleware{
		relaxed:  relaxed,
		required: required,
	}
}

type AuthMiddleware struct {
	relaxed  fiber.Handler
	required fiber.Handler
}

// Relaxed ensures that the access is authenticated when the correct credentials are provided.
// Does not forbid the access if no credentials are found.
func (r AuthMiddleware) Relaxed() func(*fiber.Ctx) error {
	return r.relaxed
}

// Required ensures that the access is authenticated.
func (r AuthMiddleware) Required() func(*fiber.Ctx) error {
	return r.required
}

func NewClientUser(u *User) *ClientUser {
	if u == nil {
		return nil
	}

	username := u.Username
	if username == "" {
		username = "Unknown"
	}

	initials := []rune(username)[0:2]

	return &ClientUser{
		Username: username,
		Initials: string(initials),
	}
}

type ClientUser struct {
	Username string `json:"username"`
	Initials string `json:"initials"`
}
