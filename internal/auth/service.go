package auth

import (
	"context"
	"encoding/base64"
	"log/slog"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

// User represents an authenticated user.
type User struct {
	ID    string
	Email string
}

// NewUser creates a new User.
func NewUser(id string) User {
	return User{ID: id}
}

func (u User) WithEmail(email string) User {
	u.Email = email

	return u
}

func (u User) Valid() bool {
	return strings.TrimSpace(u.ID) != ""
}

type Generator interface {
	Generate() ([]byte, error)
}

type AuthFlowService struct {
	provider Providers
	gen      Generator
	log      *slog.Logger
}

func New(gen Generator, providers Providers) *AuthFlowService {
	return &AuthFlowService{
		provider: providers,
		gen:      gen,
		log:      slog.Default(),
	}
}

func (s *AuthFlowService) AuthURL(provider string) (string, State, error) {
	p, err := s.provider.Find(provider)
	if err != nil {
		return "", State{}, aerrors.NewInvalidInputError(err, "auth-url-provider-not-found", "provider not found")
	}

	rawState, err := s.gen.Generate()
	if err != nil {
		return "", State{}, aerrors.NewUnknownError(err, "auth-url-state")
	}

	rawVerifier, err := s.gen.Generate()
	if err != nil {
		return "", State{}, aerrors.NewUnknownError(err, "auth-url-verifier")
	}

	state := State{
		Value:    base64.RawURLEncoding.EncodeToString(rawState),
		Verifier: base64.RawURLEncoding.EncodeToString(rawVerifier),
	}

	return p.GetAuthURL(state.Value, state.Verifier), state, nil
}

func (s *AuthFlowService) Authenticate(ctx context.Context, provider, authCode, verifier string) (*JWT, User, error) {
	p, err := s.provider.Find(provider)
	if err != nil {
		return nil, User{}, aerrors.NewInvalidInputError(err, "authenticate-provider-not-found", "provider not found")
	}

	token, err := p.ExchangeCode(ctx, authCode, verifier)
	if err != nil {
		return nil, User{}, aerrors.NewUnknownError(err, "exchange-code-failed")
	}

	claims, err := p.ValidateToken(ctx, token)
	if err != nil {
		return nil, User{}, aerrors.NewUnknownError(err, "validate-token-failed")
	}

	return token, User{
		ID:    claims.UserID,
		Email: claims.Email,
	}, nil
}

func (s *AuthFlowService) Revoke(ctx context.Context, token *JWT) error {
	p, err := s.provider.Find(token.Provider)
	if err != nil {
		return aerrors.NewInvalidInputError(err, "revoke-token-provider-not-found", "provider not found")
	}

	if err := p.RevokeToken(ctx, token); err != nil {
		return aerrors.NewUnknownError(err, "revoke-token-failed")
	}

	return nil
}
