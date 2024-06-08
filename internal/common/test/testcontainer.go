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
		return nil, err
	}

	if err = r.enableDebugIfRequired(ctx); err != nil {
		return nil, err
	}

	ip, err := r.container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := r.container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}

	return postgres.Connect(ctx, config.Database{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     mappedPort.Port(),
		Database: database,
	})
}

func (r *DatabaseRunner) enableDebugIfRequired(ctx context.Context) error {
	if e := log.Debug(); e.Enabled() {
		logs, err := r.container.Logs(ctx)
		if err != nil {
			return err
		}
		defer commonio.Close(logs)

		b, err := io.ReadAll(logs)
		if err != nil {
			return err
		}

		e.Msg(string(b))
	}

	return nil
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
