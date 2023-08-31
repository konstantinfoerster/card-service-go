package commonhttp

import (
	"github.com/gofiber/fiber/v2"
)

func RenderJSON(c *fiber.Ctx, data interface{}) error {
	err := c.JSON(data)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	return err
}
