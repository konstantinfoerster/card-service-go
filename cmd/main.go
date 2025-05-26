package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/loginapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/imaging"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"golang.org/x/sync/errgroup"
)

func setup() config.Config {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd += "/"

	var logLevel = new(slog.LevelVar)
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					source.File = strings.TrimPrefix(source.File, wd)
				}
			}

			return a
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts)).
		With("service", "card-service")
	slog.SetDefault(logger)

	var configPath string
	flag.StringVar(&configPath, "c", "./configs/application.yaml", "path to the configuration file")
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the configuration file")
	flag.Parse()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		panic(err)
	}

	if err := logLevel.UnmarshalText([]byte(cfg.Logging.Level)); err != nil {
		panic(err)
	}

	slog.Info("logging", slog.String("value", logLevel.Level().Level().String()))
	slog.Info("server", slog.Group("system",
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
		slog.Int("cpu", runtime.NumCPU()),
	))

	return cfg
}

func main() {
	cfg := setup()

	if err := run(cfg); err != nil {
		slog.Error("run error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx := context.Background()
	dbCon, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database %w", err)
	}
	defer aio.Close(dbCon)

	oidcProvider, err := auth.FromConfiguration(cfg.Oidc)
	if err != nil {
		return fmt.Errorf("failed to load oidc provider, %w", err)
	}

	timeSvc := auth.NewTimeService()
	authSvc := auth.New(cfg.Oidc, oidcProvider)
	detector := imaging.NewDetector()

	cardRepo := postgres.NewCardRepository(dbCon, cfg.Images)
	cardSvc := cards.NewCardService(cardRepo)

	collectRepo := postgres.NewCollectionRepository(dbCon, cfg.Images)
	collectSvc := cards.NewCollectionService(collectRepo)

	detectRep := postgres.NewDetectRepository(dbCon, cfg.Images)
	detectSvc := cards.NewDetectService(cardRepo, detectRep, detector)

	authMiddleware := web.NewAuthMiddleware(cfg.Oidc, authSvc)

	srv := web.NewServer(cfg.Server).RegisterRoutes(func(r fiber.Router) {
		r.Static("/public", "./public")

		cardsapi.DashboardRoutes(r, authMiddleware)
		cardsapi.SearchRoutes(r, authMiddleware, cardSvc)
		cardsapi.CollectionRoutes(r, authMiddleware, collectSvc)
		cardsapi.DetectRoutes(r, authMiddleware, detectSvc)

		apiV1 := r.Group("/api").Group("/v1")

		loginapi.Routes(apiV1, authMiddleware, cfg.Oidc, authSvc, timeSvc)
	})

	errg, ctx := errgroup.WithContext(context.Background())
	// start web-server
	errg.Go(func() error {
		return srv.Run(ctx)
	})

	readinessProbe := func(c *fiber.Ctx) bool {
		// TODO: implement readiness probe
		return true
	}
	livenessProbe := func(c *fiber.Ctx) bool {
		// TODO: implement liveness probe
		return true
	}
	// TODO: rethink that second server, maybe just add probes to
	// main server
	probeSrv := web.NewProbeServer(cfg.Probes, livenessProbe, readinessProbe)
	// start probe-server
	errg.Go(func() error {
		return probeSrv.Run(ctx)
	})

	return errg.Wait()
}
