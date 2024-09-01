package cardsapi_test

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

var validClaim = auth.NewClaims("myuser", "myUser")

func TestMain(m *testing.M) {
	exitVal := m.Run()

	os.Exit(exitVal)
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}
