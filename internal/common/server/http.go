package server

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type Server struct {
	app *fiber.App
	Cfg *config.Server
}

func NewHTTPServer(cfg *config.Server) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: problemjson.RespondWithProblemJSON,
	})

	app.Use(recover.New())
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: cfg.Cookie.EncryptionKey,
	}))
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language," +
			"Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOrigins:     "http://localhost:8000", // FIXME that should be configurable
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))

	return &Server{
		app: app,
		Cfg: cfg,
	}
}

func (s *Server) RegisterAPIRoutes(routes func(app fiber.Router)) *Server {
	routes(s.app)

	return s
}

func (s *Server) Test(req *http.Request) (*http.Response, error) {
	return s.app.Test(req)
}

func (s *Server) Run() error {
	return s.app.Listen(s.Cfg.Addr())
}
