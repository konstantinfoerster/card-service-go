package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/search/adapters"
	"github.com/konstantinfoerster/card-service-go/internal/search/application"
)

// SearchRoutes All search related routes.
func SearchRoutes(app fiber.Router, service application.Service) {
	app.Get("/search", adapters.SimpleSearch(service))
}
