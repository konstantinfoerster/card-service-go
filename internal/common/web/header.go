package web

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const HeaderHTMXRequest = "HX-Request"

func IsHTMX(c *fiber.Ctx) bool {
	return strings.ToLower(c.Get(HeaderHTMXRequest)) == "true"
}

func AcceptsHTML(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), fiber.MIMETextHTML)
}

