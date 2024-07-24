package web

import (
	"net/http"
	"path"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type Server struct {
	app *fiber.App
	Cfg config.Server
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func NewHTTPTestServer() *Server {
	cfg := config.Server{
		Cookie: config.Cookie{
			EncryptionKey: "01234567890123456789012345678901",
		},
		TemplateDir: path.Join(currentDir(), "../../../views"),
	}

	return NewHTTPServer(cfg)
}

func NewHTTPServer(cfg config.Server) *Server {
	engine := html.New(cfg.TemplateDir, ".gohtml")
	engine.AddFunc(
		"isLastIndex", func(index, length int) bool {
			return index+1 == length
		},
	)

	// FIXME: make body size configurable
	app := fiber.New(fiber.Config{
		Views: engine,
		// FIXME: error handler does not handle text/html requests
		ErrorHandler: RespondWithProblemJSON,
	})

	app.Use(recover.New())
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: cfg.Cookie.EncryptionKey,
	}))
	// FIXME: can be removed after switching to htmx
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language," +
			"Accept-Encoding,Connection,Access-Control-Allow-Origin",
		// FIXME: that should be configurable
		AllowOrigins:     "http://localhost:8000",
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
