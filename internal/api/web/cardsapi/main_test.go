package cardsapi_test

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

// var detectorUser = auth.User{ID: "detectorUser", Username: "detectorUser"}
// var detectCollector = cards.NewCollector(detectorUser.ID)
// var validUser = auth.User{ID: "myUser", Username: "myUser"}
// var collector = cards.NewCollector(validUser.ID)

var validClaim = auth.NewClaims("valid-id", "validator")
var invalidClaim = auth.NewClaims("invalid-id", "invalidator")

func TestMain(m *testing.M) {
	exitVal := m.Run()

	os.Exit(exitVal)
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}
