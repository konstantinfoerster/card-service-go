package dashboard

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
)

func Routes(r fiber.Router, cfg config.Oidc, oidcSvc oidc.UserService) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(oidcSvc, oidc.FromConfig(cfg), oidc.AllowUnauthorized())

	r.Get("/", relaxedAuthMiddleware, dashboard())
}

func dashboard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if web.IsHTMX(c) {
			return web.RenderPartial(c, "dashboard", nil)
		}

		return web.RenderPage(c, "dashboard", nil)
	}
}
