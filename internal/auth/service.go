package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

func NewRedirectURL(p Provider, redirectURI string) (*RedirectURL, error) {
	state, err := p.GenerateState().Encode()
	if err != nil {
		return nil, err
	}

	return &RedirectURL{
		URL:   p.GetAuthURL(state, redirectURI),
		State: state,
	}, nil
}

type RedirectURL struct {
	URL   string
	State string
}

// DecodeToken decode given base64 url encoded JSONWebToken.
func DecodeToken(value string) (*JWT, error) {
	return DecodeBase64[JWT](value)
}

type JWT struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	Type         string `json:"token_type"`
	Provider     string `json:"provider"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (t *JWT) Encode() (string, error) {
	return EncodeBase64(t)
}

type Claims struct {
	ID    string
	Email string
}

func NewClaims(id, email string) *Claims {
	return &Claims{ID: id, Email: email}
}

type Service interface {
	AuthInfo(ctx context.Context, provider string, token *JWT) (*Claims, error)
	AuthURL(provider string) (*RedirectURL, error)
	Authenticate(ctx context.Context, provider string, code string) (*Claims, *JWT, error)
	Logout(ctx context.Context, token *JWT) error
}

func New(cfg config.Oidc, provider ...Provider) Service {
	pp := make(map[string]Provider)
	for _, p := range provider {
		if p == nil {
			continue
		}

		pp[strings.ToLower(p.GetName())] = p
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

func (s *authFlowService) AuthURL(provider string) (*RedirectURL, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, err
	}

	return NewRedirectURL(p, s.cfg.RedirectURI)
}

func (s *authFlowService) Authenticate(
	ctx context.Context, provider string, authCode string) (*Claims, *JWT, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, nil, err
	}

	claims, jwtToken, err := p.ExchangeCode(ctx, authCode, s.cfg.RedirectURI)
	if err != nil {
		return nil, nil, aerrors.NewUnknownError(err, "exchange-code-failed")
	}

	return claims, jwtToken, nil
}

func (s *authFlowService) AuthInfo(ctx context.Context, provider string, token *JWT) (*Claims, error) {
	p, err := s.getProvider(provider)
	if err != nil {
		return nil, err
	}

	claims, err := p.ValidateToken(ctx, token)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "validate-token-failed")
	}

	return claims, nil
}

func (s *authFlowService) Logout(ctx context.Context, token *JWT) error {
	p, err := s.getProvider(token.Provider)
	if err != nil {
		return err
	}

	return p.RevokeToken(ctx, token.AccessToken)
}

func (s *authFlowService) getProvider(key string) (Provider, error) {
	if strings.TrimSpace(key) == "" {
		err := fmt.Errorf("provider mut not be empty")

		return nil, aerrors.NewInvalidInputError(err, "login-provider-empty", err.Error())
	}

	p, ok := s.provider[strings.ToLower(key)]
	if ok {
		return p, nil
	}

	err := fmt.Errorf("provider '%s' not supported", key)

	return nil, aerrors.NewInvalidInputError(err, "login-provider-not-supported", err.Error())
}
