package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/config"
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
		validate: func(ctx context.Context, token *JWT, clientID string) (*Claims, error) {
			return NewClaims("1", "test@localhost"), nil
		},
	}
}

func FromConfiguration(cfg config.Oidc) ([]Provider, error) {
	client := &http.Client{
		Timeout: cfg.ClientTimeout,
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
	ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*Claims, *JWT, error)
	ValidateToken(ctx context.Context, token *JWT) (*Claims, error)
	RevokeToken(ctx context.Context, token string) error
	GenerateState() State
}

type provider struct {
	client    *http.Client
	validate  func(ctx context.Context, token *JWT, clientID string) (*Claims, error)
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

func (p *provider) GetAuthURL(state string, redirectURI string) string {
	return fmt.Sprintf("%s?state=%s&client_id=%s&redirect_uri=%s&scope=%s&response_type=code&access_type=offline",
		p.authURL, url.QueryEscape(state), url.QueryEscape(p.clientID), url.QueryEscape(redirectURI),
		url.QueryEscape(p.scope))
}

func (p *provider) ValidateToken(ctx context.Context, token *JWT) (*Claims, error) {
	return p.validate(ctx, token, p.clientID)
}

func (p *provider) getToken(ctx context.Context, code string, redirectURI string) (*JWT, error) {
	resp, err := p.postRequest(ctx, p.tokenURL, url.Values{ //nolint:bodyclose
		"code":          {code},
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("code exchange request failed with %w", err)
	}
	defer aio.Close(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// TODO: error struct
		content, cErr := io.ReadAll(resp.Body)
		if cErr != nil {
			return nil, fmt.Errorf("unable to read code exchange response body %w", err)
		}

		return nil, fmt.Errorf("code exchange failed with response %s", content)
	}

	var jwtToken JWT
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jwtToken)
	if err != nil {
		return nil, fmt.Errorf("unable to decode code exchange response, due to %w", err)
	}

	jwtToken.Provider = p.name

	return &jwtToken, nil
}

func (p *provider) ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*Claims,
	*JWT, error) {
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
		return fmt.Errorf("token revoke request failed with %w", err)
	}
	defer aio.Close(resp.Body)

	if resp.StatusCode != http.StatusOK {
		content, cErr := io.ReadAll(resp.Body)
		if cErr != nil {
			return fmt.Errorf("unable to read token revoke response body %w", err)
		}

		return fmt.Errorf("token revoke endpoint return an error %s", content)
	}

	return nil
}

func (p *provider) GenerateState() State {
	return State{ID: uuid.New().String(), Provider: p.GetName()}
}

func (p *provider) postRequest(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request, %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return p.client.Do(req)
}

// DecodeState decode given base64 url encoded State.
func DecodeState(value string) (*State, error) {
	return DecodeBase64[State](value)
}

type State struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

func (s State) Encode() (string, error) {
	return EncodeBase64(s)
}
