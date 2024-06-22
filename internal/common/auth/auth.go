package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

var (
	// ErrNoUserInContext = errors.NewAuthorizationError(fmt.Errorf("no user in context"), "no-user-found")
	ErrNoUserInContext = fmt.Errorf("no user in context")
)

const UserContextKey = "userid"

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

func UserToCtx(ctx *fiber.Ctx, user *User) {
	ctx.Locals(UserContextKey, user)
}
