package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service/api/handlers"
	"github.com/konstantinfoerster/card-service/internal/search/service"
)

// SearchRoutes all search related routes
func SearchRoutes(app fiber.Router, service service.Service) {
	app.Get("/search", handlers.SimpleSearch(service))
}
