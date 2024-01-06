package adapter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
)

const validUser = "myUser"
const invalidUser = "invalidUser"

type fakeUserService struct {
}

var _ oidc.UserService = (*fakeUserService)(nil)

func (s *fakeUserService) GetAuthenticatedUser(_ string, token *oidc.JSONWebToken) (*auth.User, error) {
	if token.IDToken == validUser {
		return &auth.User{
			ID: validUser,
		}, nil
	}

	return nil, fmt.Errorf("invalid user")
}

func NewFakeUserService() oidc.UserService {
	return &fakeUserService{}
}

func defaultServer() *server.Server {
	return server.NewHTTPTestServer()
}

func TestMain(m *testing.M) {
	exitVal := m.Run()

	os.Exit(exitVal)
}
