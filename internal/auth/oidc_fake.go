package auth

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type ProviderOption func(*FakeProvider)

func NewFakeProvider(opts ...ProviderOption) *FakeProvider {
	p := &FakeProvider{
		name:             "testProvider",
		clientID:         "client-id",
		scope:            "openid",
		authURL:          "http://localhost/auth",
		claims:           []*Claims{},
		loggedIn:         []string{},
		validRedirectURI: []string{},
		tokenExpires:     0,
		stateID:          "state-0",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func WithClaims(c *Claims) ProviderOption {
	return func(p *FakeProvider) {
		p.claims = append(p.claims, c)
	}
}

func WithValidRedirectURI(redirectURL string) ProviderOption {
	return func(p *FakeProvider) {
		p.validRedirectURI = append(p.validRedirectURI, redirectURL)
	}
}

func WithExpire(timestamp int64) ProviderOption {
	return func(p *FakeProvider) {
		p.tokenExpires = timestamp
	}
}
func WithStateID(stateID string) ProviderOption {
	return func(p *FakeProvider) {
		p.stateID = stateID
	}
}

func WithProviderCfg(cfg config.Provider) ProviderOption {
	return func(p *FakeProvider) {
		p.clientID = cfg.ClientID
		p.scope = cfg.Scope
		p.authURL = cfg.AuthURL
	}
}

type FakeProvider struct {
	name             string
	clientID         string
	scope            string
	authURL          string
	claims           []*Claims
	validRedirectURI []string
	loggedIn         []string
	tokenExpires     int64
	stateID          string
}

func (p *FakeProvider) GetName() string {
	return p.name
}

func (p *FakeProvider) GetAuthURL(state string, redirectURI string) string {
	return fmt.Sprintf("%s?state=%s&client_id=%s&redirect_uri=%s&scope=%s&response_type=code&access_type=offline",
		p.authURL, url.QueryEscape(state), url.QueryEscape(p.clientID), url.QueryEscape(redirectURI),
		url.QueryEscape(p.scope))
}

func (p *FakeProvider) ValidateToken(ctx context.Context, token *JWT) (*Claims, error) {
	for _, c := range p.claims {
		if generateAccessToken(c.ID) == token.AccessToken {
			return c, nil
		}
	}

	return nil, fmt.Errorf("invalid token")
}

func (p *FakeProvider) ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*Claims,
	*JWT, error) {
	if slices.Contains(p.validRedirectURI, redirectURI) {
		return nil, nil, fmt.Errorf("invalid redirectURI")
	}

	for _, c := range p.claims {
		if c.ID == authCode {
			accessToken := generateAccessToken(c.ID)
			p.loggedIn = append(p.loggedIn, accessToken)

			return c, &JWT{
				AccessToken: accessToken,
				Provider:    p.name,
				ExpiresIn:   p.tokenExpires,
			}, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid authCode")
}

func generateAccessToken(id string) string {
	return id + "-accesstoken"
}

func (p *FakeProvider) RevokeToken(ctx context.Context, token string) error {
	toDelete := -1
	for i, l := range p.loggedIn {
		if l == token {
			toDelete = i
		}
	}

	if toDelete == -1 {
		return fmt.Errorf("token %s for revoke not found", token)
	}

	dd := p.loggedIn
	dd[toDelete] = dd[len(dd)-1]
	p.loggedIn = dd[:len(dd)-1]

	return nil
}

func (p *FakeProvider) GenerateState() State {
	return State{ID: p.stateID, Provider: p.GetName()}
}

// Token Returns a valid token for the given user ID.
func (p *FakeProvider) Token(userID string) *JWT {
	return &JWT{
		Provider:    p.name,
		ExpiresIn:   p.tokenExpires,
		AccessToken: generateAccessToken(userID),
	}
}
