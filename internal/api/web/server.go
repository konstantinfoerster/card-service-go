package web

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"path"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	app     *fiber.App
	Cfg     config.Server
	running atomic.Bool
}

func NewTestServer() *Server {
	_, cf, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
	}
	currentDir := path.Join(path.Dir(cf))

	cfg := config.Server{
		Cookie: config.Cookie{
			EncryptionKey: "01234567890123456789012345678901",
		},
		TemplateDir: path.Join(currentDir, "../../../views"),
	}

	return NewServer(cfg)
}

func NewProbeServer(cfg config.Server,
	livenessProbe healthcheck.HealthChecker,
	readinessProbe healthcheck.HealthChecker,
) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe:    livenessProbe,
		LivenessEndpoint: "/livez",
		ReadinessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		ReadinessEndpoint: "/readyz",
	}))

	return &Server{
		app: app,
		Cfg: cfg,
	}
}

func NewServer(cfg config.Server) *Server {
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

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
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
	noTimeout := -1

	return s.app.Test(req, noTimeout)
}

func (s *Server) Run(ctx context.Context) error {
	addr := s.Cfg.Addr()

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		nCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		<-nCtx.Done()

		return s.shutdown(ctx)
	})

	errg.Go(func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		s.app.Hooks().OnListen(func(ld fiber.ListenData) error {
			log.Info().Msgf("starting server at %s", addr)

			s.running.Store(true)

			return nil
		})
		s.app.Hooks().OnShutdown(func() error {
			s.running.Store(false)

			return nil
		})

		if s.Cfg.TLS.Enabled {
			if err := s.app.ListenTLS(addr, s.Cfg.TLS.CertFile, s.Cfg.TLS.KeyFile); err != nil {
				return fmt.Errorf("failed to start server at %s, %w", addr, err)
			}
		}
		if err := s.app.Listen(addr); err != nil {
			return fmt.Errorf("failed to start server at %s, %w", addr, err)
		}

		log.Info().Msgf("stopped listening at %s", addr)

		return nil
	})

	return errg.Wait()
}

func (s *Server) shutdown(ctx context.Context) error {
	addr := s.Cfg.Addr()
	log.Info().Msgf("shutdown server at %s, waiting to close open connections ...", addr)

	timeout := time.Second * time.Duration(15)
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, timeout)
	defer shutdownCancel()

	if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Msgf("shutdown server at %s failed, %v", addr, err)

		return err
	}

	log.Info().Msgf("shutdown server at %s successfully", addr)

	return nil
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}
