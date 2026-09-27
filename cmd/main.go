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
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"golang.org/x/sync/errgroup"
)

type arrayFlag []string

func (a *arrayFlag) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *arrayFlag) Set(value string) error {
	*a = append(*a, value)

	return nil
}

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

	var configPaths arrayFlag
	flag.Var(&configPaths, "config", "path to the configuration files e.g. --config /config.yaml --config /secret.yaml")
	flag.Parse()

	if len(configPaths) == 0 {
		configPaths = append(configPaths, "configs/application.yaml")
	}

	cfg, err := config.ReadConfigs(configPaths...)
	if err != nil {
		panic(err)
	}

	if err := logLevel.UnmarshalText([]byte(cfg.Logging.Level)); err != nil {
		panic(err)
	}

	if cfg.Server.Debug {
		if err = os.MkdirAll(cfg.Server.DebugDir, 0700); err != nil {
			panic(err)
		}
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
		return fmt.Errorf("failed to connect to database, %w", err)
	}
	defer aio.Close(dbCon)

	migrator := postgres.NewMigrator(dbCon, postgres.Scripts)
	if err := migrator.Run(ctx); err != nil {
		return fmt.Errorf("failed to run migration scripts, %w", err)
	}

	cardRepo := postgres.NewCardRepository(dbCon, cfg.Images)
	cardSvc := cards.NewCardService(cardRepo)

	collectRepo := postgres.NewCollectionRepository(dbCon, cfg.Images)
	collectSvc := cards.NewCollectionService(collectRepo)

	detectRep := postgres.NewDetectRepository(dbCon, cfg.Images)
	matcher := detection.NewMatcher(detectRep, detection.NewHasher())
	detectSvc := cards.NewDetectService(cardRepo, matcher)

	srv, err := web.NewServer(cfg.Server)
	if err != nil {
		return fmt.Errorf("failed to create web-server, %w", err)
	}

	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.DashboardRoutes(r, cfg.Auth)
		cardsapi.SearchRoutes(r, cfg.Auth, cardSvc)
		cardsapi.CollectionRoutes(r, cfg.Auth, collectSvc)
		cardsapi.DetectRoutes(r, cfg.Auth, cfg.Server, detectSvc)
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
