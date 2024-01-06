package adapter

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
)

func DashboardRoutes(r fiber.Router, cfg config.Oidc, oidcSvc oidc.UserService) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(oidcSvc, oidc.FromConfig(cfg), oidc.AllowEmptyCookie())

	r.Get("/", relaxedAuthMiddleware, dashboard())
}

func dashboard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return commonhttp.RenderLayout(c, "dashboard", nil)
	}
}
