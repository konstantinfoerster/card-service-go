package adapters

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
)

func GetCard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := oidc.UserFromCtx(c)
		if err != nil {
			return problemjson.RespondWithProblemJSON(err, c)
		}

		cardID := c.Params("id")

		if err = c.JSON(fiber.Map{"id": cardID, "user": user.ID}); err != nil {
			return problemjson.RespondWithProblemJSON(fmt.Errorf("failed to encode card result, %w", err), c)
		}

		return nil
	}
}
