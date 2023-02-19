package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

var (
	ErrNoUserInContext = common.NewAuthorizationError(fmt.Errorf("no user in context"), "no-user-found")
)

const UserContextKey = "userid"

type User struct {
	ID       string
	Username string
}

func UserFromCtx(ctx *fiber.Ctx) (*User, error) {
	u, ok := ctx.Locals(UserContextKey).(*User)
	if ok {
		return u, nil
	}

	return &User{}, ErrNoUserInContext
}
