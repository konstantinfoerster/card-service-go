package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"golang.org/x/sync/errgroup"
)

var (
	ErrInitServer = errors.New("failed to initialize server")
)

//go:embed templates/* assets/*
var embeddedFiles embed.FS

// publicPrefix is the route under which the static assets are served.
const publicPrefix = "/public"

type Server struct {
	app     *fiber.App
	Cfg     Config
	log     *slog.Logger
	running atomic.Bool
}

func NewTestServer() *Server {
	cfg := Config{}

	srv, err := NewServer(cfg)
	if err != nil {
		panic(err)
	}

	return srv
}

func NewProbeServer(cfg Config,
	livenessProbe healthcheck.HealthChecker,
	readinessProbe healthcheck.HealthChecker,
) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             cfg.MaxBodySize,
	})
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe:     livenessProbe,
		LivenessEndpoint:  "/livez",
		ReadinessProbe:    readinessProbe,
		ReadinessEndpoint: "/readyz",
	}))

	return &Server{
		app: app,
		Cfg: cfg,
		log: slog.Default(),
	}
}

func NewServer(cfg Config) (*Server, error) {
	tmplFS, err := fs.Sub(embeddedFiles, "templates")
	if err != nil {
		return nil, errors.Join(err, ErrInitServer)
	}
	static, err := newStaticAssets(cfg.Mode)
	if err != nil {
		return nil, errors.Join(err, ErrInitServer)
	}

	engine := html.NewFileSystem(http.FS(tmplFS), ".gohtml")
	engine.AddFuncMap(map[string]any{
		"isLastIndex": func(index, length int) bool {
			return index+1 == length
		},
		"asset": static.url,
	})

	// avoid 0 and negative values, 0 = unlimited
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = DefaultMaxBodySize
	}

	app := fiber.New(fiber.Config{
		Views:     engine,
		BodyLimit: cfg.MaxBodySize,
		// FIXME: error handler does not handle text/html requests
		ErrorHandler: RespondWithProblemJSON,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: false,
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

	for _, handler := range static.handlers {
		app.Use(publicPrefix, handler)
	}

	return &Server{
		app: app,
		Cfg: cfg,
		log: slog.Default(),
	}, nil
}

// staticAssets defines how asset URLs are built and which handlers serve the files.
type staticAssets struct {
	url      func(name string) (string, error)
	handlers []fiber.Handler
}

func newStaticAssets(mode Mode) (staticAssets, error) {
	if mode == dev {
		return devStaticAssets(), nil
	}

	return embeddedStaticAssets()
}

// devStaticAssets serves files from disk, they can change at any time,
// so they are not fingerprinted, but must exist.
func devStaticAssets() staticAssets {
	dir := "./internal/api/web/assets"
	assetsFS := os.DirFS(dir)

	return staticAssets{
		url: func(name string) (string, error) {
			if _, err := fs.Stat(assetsFS, name); err != nil {
				return "", fmt.Errorf("unknown asset %q: %w", name, err)
			}

			return publicPrefix + "/" + name, nil
		},
		handlers: []fiber.Handler{
			filesystem.New(filesystem.Config{Root: http.Dir(dir)}),
		},
	}
}

// embeddedStaticAssets serves the embedded files with fingerprinted URLs.
func embeddedStaticAssets() (staticAssets, error) {
	assetsFS, err := fs.Sub(embeddedFiles, "assets")
	if err != nil {
		return staticAssets{}, err
	}
	assets, err := NewAssets(assetsFS, publicPrefix)
	if err != nil {
		return staticAssets{}, err
	}

	return staticAssets{
		url: assets.Fingerprinted,
		handlers: []fiber.Handler{
			NewFingerprintMiddleware(assets),
			NewStaticFilesMiddleware(assetsFS, publicPrefix),
		},
	}, nil
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
			s.log.Info("started listening", slog.String("addr", addr))

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

		s.log.Info("stopped listening", slog.String("addr", addr))

		return nil
	})

	return errg.Wait()
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}

func (s *Server) shutdown(ctx context.Context) error {
	addr := s.Cfg.Addr()
	s.log.Info("shutdown server waiting to close open connections ...", slog.String("addr", addr))

	timeout := time.Second * time.Duration(15)
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, timeout)
	defer shutdownCancel()

	if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
		s.log.Error("shutdown server failed", slog.String("addr", addr), slog.Any("error", err))

		return err
	}

	s.log.Info("shutdown server successfully", slog.String("addr", addr))

	return nil
}
