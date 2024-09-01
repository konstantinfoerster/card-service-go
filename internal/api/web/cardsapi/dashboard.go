package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
)

func DashboardRoutes(r fiber.Router, auth web.AuthMiddleware) {
	r.Get("/", auth.Relaxed(), dashboard())
}

func dashboard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if web.IsHTMX(c) {
			return web.RenderPartial(c, "dashboard", nil)
		}

		return web.RenderPage(c, "dashboard", nil)
	}
}
