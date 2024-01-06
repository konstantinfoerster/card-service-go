package server

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
)

type Server struct {
	app *fiber.App
	Cfg *config.Server
}

func NewHTTPTestServer() *Server {
	cfg := &config.Config{
		Server: config.Server{
			Cookie: config.Cookie{
				EncryptionKey: "01234567890123456789012345678901",
			},
			TemplateDir: "../../../views",
		},
	}

	return NewHTTPServer(&cfg.Server)
}

func NewHTTPServer(cfg *config.Server) *Server {
	engine := html.New(cfg.TemplateDirOrDefault(), ".gohtml")
	engine.AddFunc(
		"isLastIndex", func(index int, len int) bool {
			return index+1 == len
		},
	)

	app := fiber.New(fiber.Config{
		Views: engine,
		// FIXME that does not handle text/html requests
		ErrorHandler: problemjson.RespondWithProblemJSON,
	})

	app.Use(recover.New())
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: cfg.Cookie.EncryptionKey,
	}))
	// FIXME can be removed after switching to htmx
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language," +
			"Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOrigins:     "http://localhost:8000", // FIXME that should be configurable
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		MaxAge:           -1,
	}))
	app.Use(favicon.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))

	return &Server{
		app: app,
		Cfg: cfg,
	}
}

func (s *Server) RegisterRoutes(routes func(app fiber.Router)) *Server {
	routes(s.app)

	return s
}

func (s *Server) Test(req *http.Request) (*http.Response, error) {
	return s.app.Test(req)
}

func (s *Server) Run() error {
	return s.app.Listen(s.Cfg.Addr())
}
