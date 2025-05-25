package postgres_test

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/rs/zerolog/log"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var connection *postgres.DBConnection
var collector = cards.NewCollector("myUser")

func TestMain(m *testing.M) {
	flag.Parse()

	ctx := context.Background()
	dbRunner := newRunner()
	if !testing.Short() {
		if err := dbRunner.Start(ctx); err != nil {
			panic(err)
		}

		var err error
		connection, err = postgres.Connect(ctx, dbRunner.Config())
		if err != nil {
			panic(err)
		}
	}

	code := m.Run()

	if err := dbRunner.Stop(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func newRunner() *databaseRunner {
	return &databaseRunner{}
}

type databaseRunner struct {
	container testcontainers.Container
	cfg       postgres.Config
	running   bool
}

func (r *databaseRunner) Start(ctx context.Context) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
	}

	dbDir, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(file), "testdata", "db"))
	if err != nil {
		return err
	}

	username := "tester"
	password := "tester"
	database := "cardmanager"

	// TODO: read env variables from config
	var initScriptDirPermissions int64 = 0755
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(dbDir, "01-init.sh"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/01-init.sh",
				FileMode:          initScriptDirPermissions,
			},
			{
				HostFilePath:      filepath.Join(dbDir, "02-create-tables.sql"),
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
			"POSTGRES_DB":       "postgres",
			"POSTGRES_PASSWORD": "test",
			"APP_DB_USER":       username,
			"APP_DB_PASS":       password,
			"APP_DB_NAME":       database,
		},
		AlwaysPullImage: true,
		WaitingFor:      wait.ForLog("[1] LOG:  database system is ready to accept connections"),
	}

	r.container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	if err = r.enableDebugIfRequired(ctx); err != nil {
		return err
	}

	ip, err := r.container.Host(ctx)
	if err != nil {
		return err
	}

	mappedPort, err := r.container.MappedPort(ctx, "5432")
	if err != nil {
		return err
	}

	r.running = true
	r.cfg = postgres.Config{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     mappedPort.Port(),
		Database: database,
	}

	return nil
}

func (r *databaseRunner) Stop(ctx context.Context) error {
	if !r.running {
		return nil
	}

	return r.container.Terminate(ctx)
}

func (r *databaseRunner) Config() postgres.Config {
	return r.cfg
}

func (r *databaseRunner) enableDebugIfRequired(ctx context.Context) error {
	if e := log.Debug(); e.Enabled() {
		logs, err := r.container.Logs(ctx)
		if err != nil {
			return err
		}
		defer aio.Close(logs)

		b, err := io.ReadAll(logs)
		if err != nil {
			return err
		}

		e.Msg(string(b))
	}

	return nil
}
