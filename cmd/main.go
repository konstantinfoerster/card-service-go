package main

import (
	"context"
	"flag"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/konstantinfoerster/card-service-go/api/routes"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/search/adapters"
	"github.com/konstantinfoerster/card-service-go/internal/search/application"
	"github.com/pkg/errors"
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

	ctx := context.Background()
	dbCon, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to the database")
	}

	rep := adapters.NewRepository(dbCon, cfg.Images)
	searchService := application.New(rep)
	oidcProvider, err := oidc.New(cfg.Oidc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to instantiate oidc provider")
	}

	authMiddleware := oidc.NewOauthMiddleware(cfg.Oidc, oidcProvider)

	app := fiber.New()
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: cfg.Server.Cookie.EncryptionKey,
	}))
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language," +
			"Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOrigins:     "*",
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(recover.New())

	api := app.Group("/api")
	v1 := api.Group("/v1")
	routes.LoginRoutes(v1, cfg.Oidc, oidcProvider)
	routes.SearchRoutes(v1, searchService)
	routes.CardsRoutes(v1, authMiddleware)

	if err = app.Listen(cfg.Server.Addr()); err != nil {
		if cErr := dbCon.Close(); cErr != nil {
			err = errors.Wrap(err, cErr.Error())
		}
		log.Fatal().Err(err).Send()
	}
}
