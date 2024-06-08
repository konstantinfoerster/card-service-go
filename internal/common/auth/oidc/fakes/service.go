package fakes

import (
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
)

type fakeUserService struct {
	loggedIn []auth.User
}

var _ oidc.UserService = (*fakeUserService)(nil)

func NewUserService(loggedInUser ...auth.User) oidc.UserService {
	return &fakeUserService{
		loggedIn: loggedInUser,
	}
}

func (s *fakeUserService) GetAuthenticatedUser(_ string, token *oidc.JSONWebToken) (*auth.User, error) {
	for _, u := range s.loggedIn {
		if u.ID == token.IDToken {
			return &u, nil
		}
	}

	return nil, fmt.Errorf("invalid user")
}
