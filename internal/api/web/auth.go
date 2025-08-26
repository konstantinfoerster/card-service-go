package web

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

type AuthMiddleware struct {
	relaxed  fiber.Handler
	required fiber.Handler
}

// NewAuthMiddleware provides fiber.Handler that can be used to ensure an authentciated access.
func NewAuthMiddleware(store *session.Store) AuthMiddleware {
	relaxed := auth.NewMiddleware(store, auth.AllowUnauthorized())
	required := auth.NewMiddleware(store)

	return AuthMiddleware{
		relaxed:  relaxed,
		required: required,
	}
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

type ClientUser struct {
	Username string `json:"username"`
	Initials string `json:"initials"`
}

func NewClientUser(u auth.User) *ClientUser {
	if u.ID == "" {
		return nil
	}

	username := u.Email
	if username == "" {
		username = "Unknown"
	}

	initials := []rune(username)[0:2]

	return &ClientUser{
		Username: username,
		Initials: string(initials),
	}
}
