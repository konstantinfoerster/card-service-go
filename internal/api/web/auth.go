package web

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

var (
	ErrNoUserInContext = fmt.Errorf("no user in context")
)

const UserContextKey = "userid"

func NewUser(id, username string) *User {
	return &User{ID: id, Username: username}
}

type User struct {
	ID       string
	Username string
}

func UserFromCtx(ctx *fiber.Ctx) (*User, error) {
	u, ok := ctx.Locals(UserContextKey).(*User)
	if ok && u != nil {
		return u, nil
	}

	return nil, ErrNoUserInContext
}

func NewAuthMiddleware(cfg auth.OidcConfig, svc auth.Service) AuthMiddleware {
	authFn := func(ctx *fiber.Ctx, claims *auth.Claims) {
		u := NewUser(claims.ID, claims.Email)
		ctx.Locals(UserContextKey, u)
	}

	return AuthMiddleware{
		cfg:    cfg,
		svc:    svc,
		authFn: authFn,
	}
}

type AuthMiddleware struct {
	svc    auth.Service
	authFn func(ctx *fiber.Ctx, claims *auth.Claims)
	cfg    auth.OidcConfig
}

func (r AuthMiddleware) Relaxed() func(*fiber.Ctx) error {
	return auth.NewOAuthMiddleware(r.svc, auth.WithAuthorized(r.authFn), auth.WithConfig(r.cfg), auth.AllowUnauthorized())
}
func (r AuthMiddleware) Required() func(*fiber.Ctx) error {
	return auth.NewOAuthMiddleware(r.svc, auth.WithAuthorized(r.authFn), auth.WithConfig(r.cfg))
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
