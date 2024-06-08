package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
)

func NewRedirectURL(p Provider, redirectURI string) (*RedirectURL, error) {
	stateBase64, err := NewState(p.GetName()).Encode()
	if err != nil {
		return nil, err
	}

	return &RedirectURL{
		URL:   p.GetAuthURL(stateBase64, redirectURI),
		State: stateBase64,
	}, nil
}

type RedirectURL struct {
	URL   string
	State string
}

func NewState(provider string) *State {
	return &State{ID: uuid.New().String(), Provider: provider}
}

// DecodeState decode given base64 url encoded State.
func DecodeState(value string) (*State, error) {
	return commonhttp.DecodeBase64[State](value)
}

type State struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

func (s *State) Encode() (string, error) {
	return commonhttp.EncodeBase64(s)
}

// DecodeToken decode given base64 url encoded JSONWebToken.
func DecodeToken(value string) (*JSONWebToken, error) {
	return commonhttp.DecodeBase64[JSONWebToken](value)
}

type JSONWebToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	Type         string `json:"token_type"`
	Provider     string `json:"provider"`
	ExpiresIn    int    `json:"expires_in"`
}

func (t *JSONWebToken) Encode() (string, error) {
	return commonhttp.EncodeBase64(t)
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
	Logout(token *JSONWebToken) error
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

	return NewRedirectURL(p, s.cfg.RedirectURI)
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

func (s *authFlowService) Logout(token *JSONWebToken) error {
	p, err := s.getProvider(token.Provider)
	if err != nil {
		return err
	}

	// FIXME: with timeout?
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

	err := fmt.Errorf("provider '%s' not supported", key)

	return nil, common.NewInvalidInputError(err, "login-provider-not-supported", err.Error())
}
