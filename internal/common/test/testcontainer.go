package commontest

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/rs/zerolog/log"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewRunner() *DatabaseRunner {
	return &DatabaseRunner{}
}

type DatabaseRunner struct {
	conn      *postgres.DBConnection
	container testcontainers.Container
}

func (r *DatabaseRunner) Start() (*postgres.DBConnection, error) {
	ctx := context.Background()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller")
	}

	dbDir, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(file), "testdata", "db"))
	if err != nil {
		return nil, err
	}

	username := "tester"
	password := "tester"
	database := "cardmanager"

	// TODO read env variables from config
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(dbDir, "/docker-entrypoint-initdb.d"),
		),
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

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	r.container = postgresC

	if e := log.Debug(); e.Enabled() {
		logs, err := postgresC.Logs(ctx)
		if err != nil {
			return nil, err
		}
		defer commonio.Close(logs)

		b, err := io.ReadAll(logs)
		if err != nil {
			return nil, err
		}

		e.Msg(string(b))
	}

	ip, err := postgresC.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}

	dbConfig := config.Database{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     mappedPort.Port(),
		Database: database,
	}

	return postgres.Connect(ctx, dbConfig)
}

func (r *DatabaseRunner) Stop() error {
	ctx := context.Background()

	return r.container.Terminate(ctx)
}

func (r *DatabaseRunner) Run(t *testing.T, runTests func(t *testing.T)) {
	t.Helper()

	con, err := r.Start()
	defer func(runner *DatabaseRunner) {
		cErr := runner.Stop()
		if cErr != nil {
			t.Fatalf("failed to stop container %v", err)
		}
	}(r)
	if err != nil {
		t.Fatalf("failed to start container %v", err)
	}

	r.conn = con

	runTests(t)
}

func (r *DatabaseRunner) Connection() *postgres.DBConnection {
	return r.conn
}
