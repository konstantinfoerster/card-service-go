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
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	loginadapters "github.com/konstantinfoerster/card-service-go/internal/login/adapters"
	searchadapters "github.com/konstantinfoerster/card-service-go/internal/search/adapters"
	search "github.com/konstantinfoerster/card-service-go/internal/search/application"
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
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Logging.LevelOrDefault()))
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

	rep := searchadapters.NewRepository(dbCon, cfg.Images)
	searchService := search.New(rep)

	client := &http.Client{
		Timeout: time.Second * time.Duration(5),
	}
	oidcProvider, err := oidc.FromConfiguration(cfg.Oidc, client)
	if err != nil {
		return fmt.Errorf("failed to load oidc provider, %w", err)
	}

	timeService := common.NewTimeService()
	authService := oidc.New(cfg.Oidc, oidcProvider)

	srv := server.NewHTTPServer(&cfg.Server).RegisterAPIRoutes(func(r fiber.Router) {
		v1 := r.Group("/api").Group("/v1")

		loginadapters.Routes(v1, cfg.Oidc, authService, timeService)
		searchadapters.Routes(v1, searchService)
	})

	return srv.Run()
}
