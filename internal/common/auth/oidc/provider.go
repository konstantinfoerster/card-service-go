package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
)

func TestProvider(cfg config.Provider, client *http.Client) Provider {
	return &provider{
		name:      "test",
		authURL:   cfg.AuthURL,
		tokenURL:  cfg.TokenURL,
		revokeURL: cfg.RevokeURL,
		client:    client,
		clientID:  cfg.ClientID,
		secret:    cfg.Secret,
		scope:     cfg.Scope,
		validate: func(ctx context.Context, token *JSONWebToken, clientID string) (*claims, error) {
			return &claims{
				ID:    "1",
				Email: "test@localhost",
			}, nil
		},
	}
}

func FromConfiguration(cfg config.Oidc, client *http.Client) ([]Provider, error) {
	if client == nil {
		return nil, fmt.Errorf("http client must not be nil")
	}

	if len(cfg.Provider) == 0 {
		return nil, fmt.Errorf("no provider configured")
	}

	pp := make([]Provider, 0)

	for k, v := range cfg.Provider {
		switch k {
		case "google":
			p, err := googleProvider(client)
			if err != nil {
				return nil, fmt.Errorf("failed to configure google provider %w", err)
			}

			if err = apply(p, v); err != nil {
				return nil, fmt.Errorf("failed to apply configuration for provider %s, %w", k, err)
			}

			pp = append(pp, p)
		default:
			return nil, fmt.Errorf("unsupported provider %s, only google provider is supported for now", k)
		}
	}

	return pp, nil
}

func apply(p *provider, cfg config.Provider) error {
	if cfg.AuthURL != "" {
		p.authURL = cfg.AuthURL
	}
	if cfg.TokenURL != "" {
		p.tokenURL = cfg.TokenURL
	}
	if cfg.RevokeURL != "" {
		p.revokeURL = cfg.RevokeURL
	}
	if cfg.Scope != "" {
		p.scope = cfg.Scope
	}

	if cfg.ClientID == "" {
		return fmt.Errorf("provider %s, client id must not be empty", p.name)
	}
	p.clientID = cfg.ClientID

	if cfg.Secret == "" {
		return fmt.Errorf("provider %s, secret must not be empty", p.name)
	}
	p.secret = cfg.Secret

	return nil
}

type Provider interface {
	GetName() string
	GetAuthURL(state string, redirectURL string) string
	ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*claims, *JSONWebToken, error)
	ValidateToken(ctx context.Context, token *JSONWebToken) (*claims, error)
	RevokeToken(ctx context.Context, token string) error
}

type provider struct {
	client    *http.Client
	validate  func(ctx context.Context, token *JSONWebToken, clientID string) (*claims, error)
	name      string
	authURL   string
	tokenURL  string
	revokeURL string
	clientID  string
	secret    string
	scope     string
}

func (p *provider) GetName() string {
	return p.name
}

func (p *provider) GetAuthURL(state string, redirectURL string) string {
	return fmt.Sprintf("%s?state=%s&client_id=%s&redirect_uri=%s&scope=%s&response_type=code&access_type=offline",
		p.authURL, url.QueryEscape(state), url.QueryEscape(p.clientID), url.QueryEscape(redirectURL),
		url.QueryEscape(p.scope))
}

func (p *provider) ValidateToken(ctx context.Context, token *JSONWebToken) (*claims, error) {
	return p.validate(ctx, token, p.clientID)
}

func (p *provider) getToken(ctx context.Context, code string, redirectURI string) (*JSONWebToken, error) {
	resp, err := p.postRequest(ctx, p.tokenURL, url.Values{ //nolint:bodyclose
		"code":          {code},
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "unable-to-execute-code-exchange-request")
	}
	defer commonio.Close(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// TODO: error struct
		content, cErr := io.ReadAll(resp.Body)
		if cErr != nil {
			return nil, aerrors.NewUnknownError(cErr, "unable-to-read-code-exchange-error-response")
		}

		return nil, aerrors.NewUnknownError(fmt.Errorf("code exchange endpoint error response %s", content),
			"code-exchange-endpoint-respond-with-error")
	}

	var jwtToken JSONWebToken
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jwtToken)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "unable-to-decode-code-exchange-response")
	}

	jwtToken.Provider = p.name

	return &jwtToken, nil
}

func (p *provider) ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*claims,
	*JSONWebToken, error) {
	jwtToken, err := p.getToken(ctx, authCode, redirectURI)
	if err != nil {
		return nil, nil, err
	}

	claims, err := p.ValidateToken(ctx, jwtToken)
	if err != nil {
		return nil, nil, err
	}

	return claims, jwtToken, nil
}

func (p *provider) RevokeToken(ctx context.Context, token string) error {
	resp, err := p.postRequest(ctx, p.revokeURL, url.Values{ //nolint:bodyclose
		"token": {token},
	})
	if err != nil {
		return aerrors.NewUnknownError(err, "unable-to-execute-token-revoke-request")
	}
	defer commonio.Close(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	content, cErr := io.ReadAll(resp.Body)
	if cErr != nil {
		return aerrors.NewUnknownError(cErr, "unable-to-read-token-revoke-error-response")
	}

	return aerrors.NewUnknownError(fmt.Errorf("token revoke endpoint return an error %s", content),
		"revoke-token-endpoint-respond-with-error")
}

func (p *provider) postRequest(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request, %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return p.client.Do(req)
}
