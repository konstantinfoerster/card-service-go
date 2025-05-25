package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

var (
	ErrProviderNoConfig           = errors.New("missing provider configuration")
	ErrProviderInvalidConfig      = errors.New("invalid provider configuration")
	ErrProviderValidateToken      = errors.New("provider token validation failed")
	ErrProviderCodeExchange       = errors.New("provider code exchange failed")
	ErrProviderTokenRevoke        = errors.New("provider token revoke failed")
	ErrProviderAuthInfo           = errors.New("provider auth info failed")
	ErrProviderKeyMissing         = errors.New("missing provider key")
	ErrProviderUnsupported        = errors.New("unsupported provider")
	ErrProviderUnexpectedResponse = errors.New("provider returned unexpected response")
)

type Provider interface {
	GetName() string
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, authCode string) (Claims, *JWT, error)
	ValidateToken(ctx context.Context, token *JWT) (Claims, error)
	RevokeToken(ctx context.Context, token *JWT) error
	GenerateState() State
}

type Providers struct {
	provider map[string]Provider
}

func NewProviders(provider ...Provider) Providers {
	pp := make(map[string]Provider)
	for _, p := range provider {
		if p == nil {
			continue
		}

		pp[strings.ToLower(p.GetName())] = p
	}

	return Providers{
		provider: pp,
	}
}

func (pp Providers) Find(key string) (Provider, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrProviderKeyMissing
	}

	p, ok := pp.provider[strings.ToLower(key)]
	if ok {
		return p, nil
	}

	return nil, fmt.Errorf("%s not found, %w", key, ErrProviderUnsupported)
}

func TestProvider(cfg config.Provider, client *http.Client) OIDCProvider {
	return OIDCProvider{
		name:        "test",
		authURL:     cfg.AuthURL,
		tokenURL:    cfg.TokenURL,
		revokeURL:   cfg.RevokeURL,
		redirectURI: cfg.RedirectURI,
		client:      client,
		clientID:    cfg.ClientID,
		secret:      cfg.Secret,
		scope:       cfg.Scope,
		validate: func(ctx context.Context, token *JWT, clientID string) (Claims, error) {
			return NewClaims("1", "test@localhost"), nil
		},
	}
}

func FromConfiguration(cfg config.Oidc) (Providers, error) {
	client := &http.Client{
		Timeout: cfg.ClientTimeout,
	}

	if len(cfg.Provider) == 0 {
		return Providers{}, ErrProviderNoConfig
	}

	pp := make([]Provider, 0)

	for k, v := range cfg.Provider {
		switch k {
		case "google":
			p, err := googleProvider(client)
			if err != nil {
				return Providers{}, errors.Join(err, ErrProviderInvalidConfig)
			}

			if err = merge(&p, v); err != nil {
				return Providers{}, err
			}

			pp = append(pp, p)
		default:
			return Providers{}, fmt.Errorf("unsupported provder %s, %w", k, ErrProviderInvalidConfig)
		}
	}

	return NewProviders(pp...), nil
}

func merge(p *OIDCProvider, cfg config.Provider) error {
	if cfg.AuthURL != "" {
		p.authURL = cfg.AuthURL
	}
	if cfg.TokenURL != "" {
		p.tokenURL = cfg.TokenURL
	}
	if cfg.RevokeURL != "" {
		p.revokeURL = cfg.RevokeURL
	}
	if cfg.RedirectURI != "" {
		p.redirectURI = cfg.RedirectURI
	}
	if cfg.Scope != "" {
		p.scope = cfg.Scope
	}

	if cfg.ClientID == "" {
		return fmt.Errorf("provider %s, client id must not be empty, %w", p.name, ErrProviderInvalidConfig)
	}
	p.clientID = cfg.ClientID

	if cfg.Secret == "" {
		return fmt.Errorf("provider %s, secret must not be empty, %w", p.name, ErrProviderInvalidConfig)
	}
	p.secret = cfg.Secret

	return nil
}

type State struct {
	ID string `json:"id"`
}

type OIDCProvider struct {
	client      *http.Client
	validate    func(ctx context.Context, token *JWT, clientID string) (Claims, error)
	name        string
	authURL     string
	tokenURL    string
	revokeURL   string
	redirectURI string
	clientID    string
	secret      string
	scope       string
}

func (p OIDCProvider) GetName() string {
	return p.name
}

func (p OIDCProvider) GetAuthURL(state string) string {
	return fmt.Sprintf("%s?state=%s&client_id=%s&redirect_uri=%s&scope=%s&response_type=code&access_type=offline",
		p.authURL, url.QueryEscape(state), url.QueryEscape(p.clientID), url.QueryEscape(p.redirectURI),
		url.QueryEscape(p.scope))
}

func (p OIDCProvider) ValidateToken(ctx context.Context, token *JWT) (Claims, error) {
	c, err := p.validate(ctx, token, p.clientID)
	if err != nil {
		return Claims{}, errors.Join(err, ErrProviderValidateToken)
	}

	return c, nil
}

func (p OIDCProvider) ExchangeCode(ctx context.Context, authCode string) (Claims, *JWT, error) {
	body, err := p.postRequest(ctx, p.tokenURL, url.Values{
		"code":          {authCode},
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"redirect_uri":  {p.redirectURI},
		"grant_type":    {"authorization_code"},
	}, http.StatusOK)
	if err != nil {
		return Claims{}, nil, fmt.Errorf("post failed duo to %w", errors.Join(err, ErrProviderCodeExchange))
	}
	defer aio.Close(body)

	var jwtToken JWT
	if dErr := json.NewDecoder(body).Decode(&jwtToken); dErr != nil {
		return Claims{}, nil, fmt.Errorf("unable to decode response, %w", errors.Join(dErr, ErrProviderCodeExchange))
	}

	jwtToken.Provider = p.name

	claims, err := p.ValidateToken(ctx, &jwtToken)
	if err != nil {
		return Claims{}, nil, err
	}

	return claims, &jwtToken, nil
}

func (p OIDCProvider) RevokeToken(ctx context.Context, token *JWT) error {
	if token == nil {
		return errors.Join(errEmptyToken, ErrProviderTokenRevoke)
	}
	body, err := p.postRequest(ctx, p.revokeURL, url.Values{
		"token": {token.AccessToken},
	}, http.StatusOK)
	if err != nil {
		return fmt.Errorf("post failed duo to %w", errors.Join(err, ErrProviderTokenRevoke))
	}
	defer aio.Close(body)

	return nil
}

func (p OIDCProvider) GenerateState() State {
	return State{ID: uuid.New().String()}
}

func (p OIDCProvider) postRequest(ctx context.Context, url string, data url.Values,
	expectedStatus int) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create post request, %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed with, %w", err)
	}

	if resp.StatusCode != expectedStatus {
		defer aio.Close(resp.Body)
		// TODO: decode into error struct
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("expected status %d but got %d, failed to read error response, %w",
				expectedStatus, resp.StatusCode, err)
		}

		return nil, fmt.Errorf("expected status %d but got %d due to %s, %w",
			expectedStatus, resp.StatusCode, content, ErrProviderUnexpectedResponse)
	}

	return resp.Body, nil
}
