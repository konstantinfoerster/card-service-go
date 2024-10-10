package auth

import (
	"context"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

func NewRedirectURL(p Provider, state string) (RedirectURL, error) {
	return RedirectURL{
		URL:   p.GetAuthURL(state),
		State: state,
	}, nil
}

type RedirectURL struct {
	URL   string
	State string
}

// DecodeSession decode given base64 url encoded JSONWebToken.
func DecodeSession(value string) (*JWT, error) {
	jwt, err := decodeBase64[JWT](value)
	if err != nil {
		return nil, aerrors.NewInvalidInputError(err, "invalid-session", "invalid session value")
	}

	return jwt, nil
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
	s, err := encodeBase64(t)
	if err != nil {
		return "", aerrors.NewUnknownError(err, "unable-to-encode-token")
	}

	return s, nil
}

type Claims struct {
	ID    string
	Email string
}

func NewClaims(id, email string) Claims {
	return Claims{ID: id, Email: email}
}

type Service interface {
	AuthInfo(ctx context.Context, provider string, token *JWT) (Claims, error)
	AuthURL(provider string) (RedirectURL, error)
	Authenticate(ctx context.Context, provider string, code string) (Claims, *JWT, error)
	Logout(ctx context.Context, token *JWT) error
}

func New(cfg config.Oidc, providers Providers) Service {
	return &authFlowService{
		provider: providers,
		cfg:      cfg,
	}
}

type authFlowService struct {
	provider Providers
	cfg      config.Oidc
}

func (s *authFlowService) AuthURL(provider string) (RedirectURL, error) {
	p, err := s.provider.Find(provider)
	if err != nil {
		return RedirectURL{}, aerrors.NewInvalidInputError(err, "auth-url-provider-not-found", "provider not found")
	}

	encodedState, err := encodeBase64(p.GenerateState())
	if err != nil {
		return RedirectURL{}, aerrors.NewInvalidInputError(err, "invalid-state", "invalid state value")
	}

	return NewRedirectURL(p, encodedState)
}

func (s *authFlowService) Authenticate(ctx context.Context, provider string, authCode string) (Claims, *JWT, error) {
	p, err := s.provider.Find(provider)
	if err != nil {
		return Claims{}, nil, aerrors.NewInvalidInputError(err, "authenticate-provider-not-found", "provider not found")
	}

	claims, jwtToken, err := p.ExchangeCode(ctx, authCode)
	if err != nil {
		return Claims{}, nil, aerrors.NewUnknownError(err, "exchange-code-failed")
	}

	return claims, jwtToken, nil
}

func (s *authFlowService) AuthInfo(ctx context.Context, provider string, token *JWT) (Claims, error) {
	p, err := s.provider.Find(provider)
	if err != nil {
		return Claims{}, aerrors.NewInvalidInputError(err, "auth-info-provider-not-found", "provider not found")
	}

	claims, err := p.ValidateToken(ctx, token)
	if err != nil {
		return Claims{}, aerrors.NewUnknownError(err, "validate-token-failed")
	}

	return claims, nil
}

func (s *authFlowService) Logout(ctx context.Context, token *JWT) error {
	p, err := s.provider.Find(token.Provider)
	if err != nil {
		return aerrors.NewInvalidInputError(err, "revoke-token-provider-not-found", "provider not found")
	}

	if err := p.RevokeToken(ctx, token); err != nil {
		return aerrors.NewUnknownError(err, "revoke-token-failed")
	}

	return nil
}
