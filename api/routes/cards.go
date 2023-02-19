package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards/adapters"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
)

// CardsRoutes All card related routes.
func CardsRoutes(app fiber.Router, m *oidc.Middleware) {
	app.Get("/cards/:id", m.Middleware(), adapters.GetCard())
}
