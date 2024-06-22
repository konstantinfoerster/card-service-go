//go:build opencv

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/clock"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/dashboard"
	"github.com/konstantinfoerster/card-service-go/internal/login"
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
	dbCon, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database %w", err)
	}
	defer commonio.Close(dbCon)

	client := &http.Client{
		Timeout: time.Second * time.Duration(5),
	}
	oidcProvider, err := oidc.FromConfiguration(cfg.Oidc, client)
	if err != nil {
		return fmt.Errorf("failed to load oidc provider, %w", err)
	}

	timeService := clock.NewTimeService()
	authService := oidc.New(cfg.Oidc, oidcProvider)
	detector := detect.NewDetector()
	hasher := detect.NewPHasher()

	searchRepo := cards.NewRepository(dbCon, cfg.Images)
	searchService := cards.NewService(searchRepo)

	collectionRep := collection.NewRepository(dbCon, cfg.Images)
	collectService := collection.NewService(collectionRep, searchRepo)

	detectRep := detect.NewRepository(dbCon, cfg.Images)
	detectService := detect.NewDetectService(detectRep, detector, hasher)

	srv := web.NewHTTPServer(&cfg.Server).RegisterRoutes(func(r fiber.Router) {
		r.Static("/public", "./public")

		dashboard.Routes(r, cfg.Oidc, authService)
		cards.Routes(r, cfg.Oidc, authService, searchService)
		collection.Routes(r, cfg.Oidc, authService, collectService)
		detect.Routes(r, cfg.Oidc, authService, detectService)

		apiV1 := r.Group("/api").Group("/v1")

		login.Routes(apiV1, cfg.Oidc, authService, timeService)
	})

	return srv.Run()
}
