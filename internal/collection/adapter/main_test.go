package adapter_test

import (
	"os"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
)

var validUser = auth.User{ID: "myUser", Username: "myUser"}

const invalidUser = "invalidUser"

func defaultServer() *server.Server {
	return server.NewHTTPTestServer()
}

func TestMain(m *testing.M) {
	exitVal := m.Run()

	os.Exit(exitVal)
}
