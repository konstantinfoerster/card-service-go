package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/login/adapters"
)

// LoginRoutes All login and user related routes.
func LoginRoutes(app fiber.Router, cfg config.Oidc, sp *oidc.SupportedProvider) {
	app.Get("/login/:provider", adapters.Login(cfg.RedirectURI, sp))
	app.Post("/login/:provider/token", adapters.ExchangeCode(cfg.RedirectURI, sp))
	app.Post("/logout", adapters.Logout(sp))
	app.Get("/user", adapters.GetUser())
}
