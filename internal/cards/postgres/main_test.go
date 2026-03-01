package postgres_test

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var connection *postgres.DBConnection
var migrationCfg postgres.Config
var collector = cards.NewCollector("myUser")

func TestMain(m *testing.M) {
	flag.Parse()

	ctx := context.Background()
	dbRunner := newRunner()
	if !testing.Short() {
		info, err := dbRunner.Start(ctx)
		if err != nil {
			panic(err)
		}

		appCfg := postgres.Config{
			Username: "tester",
			Password: "tester",
			Host:     info.Host,
			Port:     info.Port,
			Database: "cardmanager",
		}
		connection = Connect(ctx, appCfg)

		migrationCfg = postgres.Config{
			Username: "migration",
			Password: "migration",
			Host:     info.Host,
			Port:     info.Port,
			Database: "migration",
		}
	}

	code := m.Run()

	if err := dbRunner.Stop(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

type logConsumer struct{}

func (lc *logConsumer) Accept(l testcontainers.Log) {
	slog.Debug(string(l.Content))
}

type ConnectionInfo struct {
	Host string
	Port string
}

type databaseRunner struct {
	container testcontainers.Container
	running   bool
}

func newRunner() *databaseRunner {
	return &databaseRunner{}
}

func (r *databaseRunner) Start(ctx context.Context) (ConnectionInfo, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
	}

	scriptsDir, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(file), "scripts"))
	if err != nil {
		return ConnectionInfo{}, err
	}

	dbDir, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(file), "testdata", "db"))
	if err != nil {
		return ConnectionInfo{}, err
	}

	appUser := "tester"
	appPassword := "tester"
	appDatabase := "cardmanager"

	migrationUser := "migration"
	migrationPassword := migrationUser
	migrationDatabase := migrationUser

	// TODO: read env variables from config
	var initScriptDirPermissions int64 = 0755
	req := testcontainers.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(dbDir, "01-init.sh"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/01-init.sh",
				FileMode:          initScriptDirPermissions,
			},
			{
				HostFilePath:      filepath.Join(scriptsDir, "001-create-tables.sql"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/02-create-tables.sql",
				FileMode:          initScriptDirPermissions,
			},
			{
				HostFilePath:      filepath.Join(dbDir, "03-data.sql"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/03-data.sql",
				FileMode:          initScriptDirPermissions,
			},
		},
		Env: map[string]string{
			"POSTGRES_DB":        "postgres",
			"POSTGRES_USER":      "postgres",
			"POSTGRES_PASSWORD":  "test",
			"APP_DB_USER":        appUser,
			"APP_DB_PASS":        appPassword,
			"APP_DB_NAME":        appDatabase,
			"MIGRATION_USER":     migrationUser,
			"MIGRATION_PASSWORD": migrationPassword,
			"MIGRATION_DATABASE": migrationDatabase,
		},
		AlwaysPullImage: true,
		WaitingFor:      wait.ForLog("[1] LOG:  database system is ready to accept connections"),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts: []testcontainers.LogProductionOption{
				testcontainers.WithLogProductionTimeout(10 * time.Second),
			},
			Consumers: []testcontainers.LogConsumer{&logConsumer{}},
		},
	}

	r.container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return ConnectionInfo{}, err
	}

	ip, err := r.container.Host(ctx)
	if err != nil {
		return ConnectionInfo{}, err
	}

	mappedPort, err := r.container.MappedPort(ctx, "5432")
	if err != nil {
		return ConnectionInfo{}, err
	}

	r.running = true

	return ConnectionInfo{
		Host: ip,
		Port: mappedPort.Port(),
	}, nil
}

func (r *databaseRunner) Stop(ctx context.Context) error {
	if !r.running {
		return nil
	}

	return r.container.Terminate(ctx)
}

func Connect(ctx context.Context, cfg postgres.Config) *postgres.DBConnection {
	connection, err := postgres.Connect(ctx, cfg)
	if err != nil {
		panic(err)
	}

	return connection
}
