package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aio"
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
	GetAuthURL(state, verifier string) string
	ExchangeCode(ctx context.Context, authCode, verifier string) (*JWT, error)
	ValidateToken(ctx context.Context, token *JWT) (Claim, error)
	RevokeToken(ctx context.Context, token *JWT) error
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

func FromConfiguration(cfg Config) (Providers, error) {
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

func merge(p *OIDCProvider, cfg ProviderCfg) error {
	if cfg.AuthURL != "" {
		p.AuthURL = cfg.AuthURL
	}
	if cfg.TokenURL != "" {
		p.TokenURL = cfg.TokenURL
	}
	if cfg.RevokeURL != "" {
		p.RevokeURL = cfg.RevokeURL
	}
	if cfg.RedirectURI != "" {
		p.RedirectURI = cfg.RedirectURI
	}
	if len(cfg.Scopes) > 0 {
		p.Scopes = cfg.Scopes
	}

	if cfg.ClientID == "" {
		return fmt.Errorf("provider %s, client id must not be empty, %w", p.Name, ErrProviderInvalidConfig)
	}
	p.ClientID = cfg.ClientID

	if cfg.Secret == "" {
		return fmt.Errorf("provider %s, secret must not be empty, %w", p.Name, ErrProviderInvalidConfig)
	}
	p.Secret = cfg.Secret

	return nil
}

type State struct {
	Value    string `json:"value"`
	Verifier string `json:"verifier"`
}

type OIDCProvider struct {
	Client      *http.Client
	Validate    func(ctx context.Context, token *JWT, clientID string) (Claim, error)
	Name        string
	AuthURL     string
	TokenURL    string
	RevokeURL   string
	RedirectURI string
	ClientID    string
	Secret      string
	Scopes      []string
}

func (p OIDCProvider) GetName() string {
	return p.Name
}

func (p OIDCProvider) GetAuthURL(state, verifier string) string {
	sha := sha256.Sum256([]byte(verifier))
	cc := base64.RawURLEncoding.EncodeToString(sha[:])

	v := url.Values{
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {cc},
		"client_id":             {p.ClientID},
		"redirect_uri":          {p.RedirectURI},
		"scope":                 {strings.Join(p.Scopes, " ")},
		"response_type":         {"code"},
		"access_type":           {"online"},
	}

	return p.AuthURL + "?" + v.Encode()
}

func (p OIDCProvider) ValidateToken(ctx context.Context, token *JWT) (Claim, error) {
	c, err := p.Validate(ctx, token, p.ClientID)
	if err != nil {
		return Claim{}, errors.Join(err, ErrProviderValidateToken)
	}

	return c, nil
}

func (p OIDCProvider) ExchangeCode(ctx context.Context, authCode, verifier string) (*JWT, error) {
	body, err := p.postRequest(ctx, p.TokenURL, url.Values{
		"code":          {authCode},
		"code_verifier": {verifier},
		"client_id":     {p.ClientID},
		"client_secret": {p.Secret},
		"redirect_uri":  {p.RedirectURI},
		"grant_type":    {"authorization_code"},
	}, http.StatusOK)
	defer aio.Close(body)
	if err != nil {
		return nil, fmt.Errorf("post failed duo to %w", errors.Join(err, ErrProviderCodeExchange))
	}

	var jwtToken JWT
	if dErr := json.NewDecoder(body).Decode(&jwtToken); dErr != nil {
		return nil, fmt.Errorf("unable to decode response, %w", errors.Join(dErr, ErrProviderCodeExchange))
	}

	jwtToken.Provider = p.Name

	return &jwtToken, nil
}

func (p OIDCProvider) RevokeToken(ctx context.Context, token *JWT) error {
	if token == nil {
		return errors.Join(errEmptyToken, ErrProviderTokenRevoke)
	}
	body, err := p.postRequest(ctx, p.RevokeURL, url.Values{
		"token": {token.AccessToken},
	}, http.StatusOK)
	defer aio.Close(body)
	if err != nil {
		return fmt.Errorf("post failed duo to %w", errors.Join(err, ErrProviderTokenRevoke))
	}

	return nil
}

func (p OIDCProvider) postRequest(ctx context.Context, url string, data url.Values,
	expectedStatus int) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create post request, %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.Client.Do(req)
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
