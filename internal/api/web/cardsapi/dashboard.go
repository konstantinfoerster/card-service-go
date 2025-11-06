package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
)

func DashboardRoutes(r fiber.Router, cfg web.Auth) {
	authHandler := web.NewMiddleware(cfg, web.AllowUnauthorized())

	r.Get("/", authHandler, dashboard(cfg))
}

func dashboard(cfg web.Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if web.IsHTMX(c) {
			return web.RenderPartial(c, "dashboard", nil)
		}

		return web.RenderPage(c, cfg, "dashboard", nil)
	}
}
