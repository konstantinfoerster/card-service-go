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
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
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

	// avoid 0 and negative values, fiber would silently fall back to its 4MB default
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = DefaultMaxBodySize
	}

	app := fiber.New(fiber.Config{
		Views:     engine,
		BodyLimit: cfg.MaxBodySize,
		// FIXME: error handler does not handle text/html requests
		ErrorHandler: RespondWithProblemJSON,
	})

	app.Use(fiberrecover.New(fiberrecover.Config{
		EnableStackTrace: false,
	}))
	app.Use(favicon.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${requestPath}\n",
		CustomTags: map[string]logger.LogFunc{
			"requestPath": logRequestPath,
		},
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

// logRequestPath writes the path as requested by the client without
// query parameters to avoid logging sensitive data,
// fingerprint middleware rewrites the path, but we need the original one.
func logRequestPath(output logger.Buffer, c *fiber.Ctx, _ *logger.Data, _ string) (int, error) {
	path, _, _ := strings.Cut(c.OriginalURL(), "?")

	return output.WriteString(path)
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
			info, err := fs.Stat(assetsFS, name)
			if err != nil {
				return "", fmt.Errorf("asset %q: %w: %w", name, ErrUnknownAsset, err)
			}
			if info.IsDir() {
				return "", fmt.Errorf("asset %q is a directory: %w", name, ErrUnknownAsset)
			}

			return publicPrefix + "/" + name, nil
		},
		handlers: []fiber.Handler{
			func(c *fiber.Ctx) error {
				// no cache for dev
				c.Set(fiber.HeaderCacheControl, "no-cache")

				return c.Next()
			},
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

// Run serves requests until ctx is canceled, then shuts the server down gracefully.
func (s *Server) Run(ctx context.Context) error {
	addr := s.Cfg.Addr()
	s.app.Hooks().OnListen(func(fiber.ListenData) error {
		s.log.Info("started listening", slog.String("addr", addr))
		s.running.Store(true)

		return nil
	})
	defer s.running.Store(false)

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		<-ctx.Done()
		if !s.IsRunning() {
			return nil
		}

		return s.shutdown()
	})
	errg.Go(func() error {
		if s.Cfg.TLS.Enabled {
			return s.app.ListenTLS(addr, s.Cfg.TLS.CertFile, s.Cfg.TLS.KeyFile)
		}

		return s.app.Listen(addr)
	})

	return errg.Wait()
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}

func (s *Server) shutdown() error {
	addr := s.Cfg.Addr()
	s.log.Info("shutdown server waiting to close open connections ...", slog.String("addr", addr))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("shutdown server at %s failed, %w", addr, err)
	}

	s.log.Info("shutdown server successfully", slog.String("addr", addr))

	return nil
}
