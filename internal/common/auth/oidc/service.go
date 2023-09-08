package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type RedirectURL struct {
	URL   string
	State string
}

type JSONWebToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	Type         string `json:"token_type"`
	Provider     string `json:"provider"`
}

type claims struct {
	ID    string
	Email string
}

func claimsToUser(claims *claims) *auth.User {
	return &auth.User{
		ID:       claims.ID,
		Username: claims.Email,
	}
}

type UserService interface {
	GetAuthenticatedUser(provider string, token *JSONWebToken) (*auth.User, error)
}

type Service interface {
	UserService
	GetAuthURL(provider string) (*RedirectURL, error)
	Authenticate(provider string, code string) (*auth.User, *JSONWebToken, error)
	Logout(provider string, token *JSONWebToken) error
}

func New(cfg config.Oidc, provider []Provider) Service {
	pp := make(map[string]Provider)
	for _, p := range provider {
		pp[p.GetName()] = p
	}

	return &authFlowService{
		provider: pp,
		cfg:      cfg,
	}
}

type authFlowService struct {
	provider map[string]Provider
	cfg      config.Oidc
}

func (s *authFlowService) GetAuthURL(provider string) (*RedirectURL, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, err
	}

	stateID := uuid.New().String()

	return &RedirectURL{
		URL:   p.GetAuthURL(stateID, s.cfg.RedirectURI),
		State: stateID,
	}, nil
}

func (s *authFlowService) Authenticate(provider string, authCode string) (*auth.User, *JSONWebToken, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	claims, jwtToken, err := p.ExchangeCode(ctx, authCode, s.cfg.RedirectURI)
	if err != nil {
		return nil, nil, err
	}

	return claimsToUser(claims), jwtToken, nil
}

func (s *authFlowService) GetAuthenticatedUser(provider string, token *JSONWebToken) (*auth.User, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, err
	}

	claims, err := p.ValidateToken(context.Background(), token)
	if err != nil {
		return nil, err
	}

	return claimsToUser(claims), nil
}

func (s *authFlowService) Logout(provider string, token *JSONWebToken) error {
	p, err := s.getProvider(provider)
	if err != nil {
		return err
	}

	// FIXME with timeout?
	ctx := context.Background()

	return p.RevokeToken(ctx, token.AccessToken)
}

func (s *authFlowService) getProvider(key string) (Provider, error) {
	if strings.TrimSpace(key) == "" {
		err := fmt.Errorf("provider mut not be empty")

		return nil, common.NewInvalidInputError(err, "login-provider-empty", err.Error())
	}

	p, ok := s.provider[strings.ToLower(key)]
	if ok {
		return p, nil
	}

	err := fmt.Errorf("provider %s not supported", key)

	return nil, common.NewInvalidInputError(err, "login-provider-not-supported", err.Error())
}
