package main

import (
	"context"
	"flag"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/konstantinfoerster/card-service/api/routes"
	"github.com/konstantinfoerster/card-service/internal/common/postgres"
	"github.com/konstantinfoerster/card-service/internal/config"
	"github.com/konstantinfoerster/card-service/internal/search/adapters"
	"github.com/konstantinfoerster/card-service/internal/search/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"os"
	"runtime"
	"strings"
	"time"
)

var cfg *config.Config

func init() {
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

	var err error
	cfg, err = config.NewConfig(configPath)
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
}

func main() {
	ctx := context.Background()
	dbCon, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to the database")
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msgf("Failed to close database connection")
		}
	}(dbCon.Close)

	rep := adapters.NewRepository(dbCon, cfg.Images)
	searchService := service.New(rep)

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip}  ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(recover.New())

	api := app.Group("/api")
	v1 := api.Group("/v1")
	routes.SearchRoutes(v1, searchService)

	log.Fatal().Err(app.Listen(cfg.Server.Addr())).Send()
}
