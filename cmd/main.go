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
	collectionadapter "github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	collection "github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	loginadapter "github.com/konstantinfoerster/card-service-go/internal/login/adapter"
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

	client := &http.Client{
		Timeout: time.Second * time.Duration(5),
	}
	oidcProvider, err := oidc.FromConfiguration(cfg.Oidc, client)
	if err != nil {
		return fmt.Errorf("failed to load oidc provider, %w", err)
	}

	timeService := common.NewTimeService()
	authService := oidc.New(cfg.Oidc, oidcProvider)

	cardRepo := collectionadapter.NewCardRepository(dbCon, cfg.Images)
	searchService := collection.NewSearchService(cardRepo)

	collectionRep := collectionadapter.NewCollectionRepository(dbCon, cfg.Images)
	collectService := collection.NewCollectService(collectionRep, cardRepo)

	srv := server.NewHTTPServer(&cfg.Server).RegisterAPIRoutes(func(r fiber.Router) {
		v1 := r.Group("/api").Group("/v1")

		loginadapter.Routes(v1, cfg.Oidc, authService, timeService)
		collectionadapter.SearchRoutes(v1, cfg.Oidc, authService, searchService)
		collectionadapter.CollectRoutes(v1, cfg.Oidc, authService, collectService)
	})

	return srv.Run()
}
