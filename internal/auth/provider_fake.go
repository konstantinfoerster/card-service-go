package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type ProviderOption func(*FakeProvider)

func WithClaims(c Claim) ProviderOption {
	return func(p *FakeProvider) {
		p.claims = append(p.claims, c)
	}
}

type FakeProvider struct {
	name        string
	clientID    string
	scopes      []string
	authURL     string
	redirectURI string
	claims      []Claim
	loggedIn    []string
}

func NewFakeProvider(opts ...ProviderOption) *FakeProvider {
	p := &FakeProvider{
		name:        "testProvider",
		clientID:    "client-id",
		scopes:      []string{"openid"},
		authURL:     "http://localhost/auth",
		redirectURI: "http://localhost/home",
		claims:      []Claim{},
		loggedIn:    []string{},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *FakeProvider) GetName() string {
	return p.name
}

func (p *FakeProvider) GetAuthURL(state, verifier string) string {
	v := url.Values{
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {verifier},
		"client_id":             {p.clientID},
		"redirect_uri":          {p.redirectURI},
		"scope":                 {strings.Join(p.scopes, " ")},
		"response_type":         {"code"},
		"access_type":           {"offline"},
	}

	return p.authURL + "?" + v.Encode()
}

func (p *FakeProvider) ValidateToken(ctx context.Context, token *JWT) (Claim, error) {
	for _, c := range p.claims {
		if generateAccessToken(c.UserID) == token.AccessToken {
			return c, nil
		}
	}

	return Claim{}, ErrProviderValidateToken
}

func (p *FakeProvider) ExchangeCode(ctx context.Context, authCode, verifier string) (*JWT, error) {
	for _, c := range p.claims {
		if c.UserID == authCode {
			accessToken := generateAccessToken(c.UserID)
			p.loggedIn = append(p.loggedIn, accessToken)

			return &JWT{
				AccessToken: accessToken,
				Provider:    p.name,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid authCode, %w", ErrProviderCodeExchange)
}

func generateAccessToken(id string) string {
	return id + "-accesstoken"
}

func (p *FakeProvider) RevokeToken(ctx context.Context, token *JWT) error {
	toDelete := -1
	for i, l := range p.loggedIn {
		if l == token.AccessToken {
			toDelete = i
		}
	}

	if toDelete == -1 {
		return fmt.Errorf("token not found %w", ErrProviderTokenRevoke)
	}

	dd := p.loggedIn
	dd[toDelete] = dd[len(dd)-1]
	p.loggedIn = dd[:len(dd)-1]

	return nil
}

var _ Provider = (*FakeProvider)(nil)
