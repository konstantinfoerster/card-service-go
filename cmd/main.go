//go:build opencv

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/loginapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	cardspg "github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/image"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func setup() *config.Config {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Stack().
		Caller().
		Logger()

	var configPath string
	flag.StringVar(&configPath, "c", "./configs/application.yaml", "path to the configuration file")
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the configuration file")
	flag.Parse()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		panic(err)
	}
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Logging.Level))
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(level)

	log.Info().Msgf("OS\t\t %s", runtime.GOOS)
	log.Info().Msgf("ARCH\t\t %s", runtime.GOARCH)
	log.Info().Msgf("CPUs\t\t %d", runtime.NumCPU())

	return cfg
}

func main() {
	cfg := setup()

	if err := run(cfg); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(cfg *config.Config) error {
	ctx := context.Background()
	dbCon, err := cardspg.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database %w", err)
	}
	defer aio.Close(dbCon)

	oidcProvider, err := auth.FromConfiguration(cfg.Oidc)
	if err != nil {
		return fmt.Errorf("failed to load oidc provider, %w", err)
	}

	timeSvc := auth.NewTimeService()
	authSvc := auth.New(cfg.Oidc, oidcProvider...)
	detector := image.NewDetector()
	hasher := image.NewPHasher()

	searchRepo := postgres.NewCardRepository(dbCon, cfg.Images)
	searchSvc := cards.NewCardService(searchRepo)

	collectionRep := postgres.NewCollectionRepository(dbCon, cfg.Images)
	collectSvc := cards.NewCollectionService(collectionRep)

	detectRep := postgres.NewDetectRepository(dbCon, cfg.Images)
	detectSvc := cards.NewDetectService(detectRep, detector, hasher)

	authMiddleware := web.NewAuthMiddleware(cfg.Oidc, authSvc)

	srv := web.NewHTTPServer(cfg.Server).RegisterRoutes(func(r fiber.Router) {
		r.Static("/public", "./public")

		cardsapi.DashboardRoutes(r, authMiddleware)
		cardsapi.SearchRoutes(r, authMiddleware, searchSvc)
		cardsapi.CollectionRoutes(r, authMiddleware, collectSvc)
		cardsapi.DetectRoutes(r, authMiddleware, detectSvc)

		apiV1 := r.Group("/api").Group("/v1")

		loginapi.Routes(apiV1, authMiddleware, cfg.Oidc, authSvc, timeSvc)
	})

	return srv.Run()
}
