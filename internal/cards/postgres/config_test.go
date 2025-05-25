package postgres_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnectionURL(t *testing.T) {
	cfg := postgres.Config{
		Host:     "localhost",
		Port:     "5432",
		Database: "test",
		Username: "Tester",
		Password: "secret",
	}

	assert.Equal(t, "postgres://Tester:secret@localhost:5432/test", cfg.ConnectionURL())
}
