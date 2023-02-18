package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/login/adapters"
)

// LoginRoutes All login and user related routes.
func LoginRoutes(app fiber.Router, cfg config.Oidc) {
	app.Get("/login/:provider", adapters.Login(cfg))
	app.Post("/login/:provider/token", adapters.ExchangeCode(cfg))
	app.Post("/logout", adapters.Logout())
	app.Get("/user", adapters.GetUser())
}
