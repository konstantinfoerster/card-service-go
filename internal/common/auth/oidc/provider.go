package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/rs/zerolog/log"
)

type JSONWebToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	Type         string `json:"token_type"`
	Provider     string `json:"provider"`
}

type Claims struct {
	ID    string
	Email string
}

func New(cfg config.Oidc) (*SupportedProvider, error) {
	client := &http.Client{
		Timeout: time.Second * time.Duration(5),
	}

	googleProvider := DefaultGoogleProviderConfiguration()
	googleProvider.Client = client
	sp := SupportedProvider{
		provider: map[string]*Provider{
			googleProvider.Name: googleProvider,
		},
	}

	if err := sp.merge(cfg.Provider); err != nil {
		return nil, err
	}

	return &sp, nil
}

type SupportedProvider struct {
	provider map[string]*Provider
}

func (sp *SupportedProvider) merge(providers map[string]config.Provider) error {
	for k, p := range providers {
		k = strings.ToLower(k)
		val, ok := sp.provider[k]
		if !ok {
			return common.NewUnknownError(fmt.Errorf("configured provider %s not supported", k), "unknown-provider")
		}

		if p.AuthURL != "" {
			val.AuthURL = p.AuthURL
		}
		if p.TokenURL != "" {
			val.TokenURL = p.TokenURL
		}
		if p.RevokeURL != "" {
			val.RevokeURL = p.RevokeURL
		}
		if p.Scope != "" {
			val.Scope = p.Scope
		}
		val.ClientID = p.ClientID
		val.Secret = p.Secret

		sp.provider[k] = val
	}

	return nil
}

func (sp *SupportedProvider) Find(key string) (*Provider, error) {
	p, ok := sp.provider[key]
	if ok {
		return p, nil
	}

	err := fmt.Errorf("provider %s not supported", key)

	return nil, common.NewInvalidInputError(err, "login-provider-not-supported", err.Error())
}

type Provider struct {
	Client    *http.Client
	Name      string
	AuthURL   string
	TokenURL  string
	RevokeURL string
	ClientID  string
	Secret    string
	Scope     string
	Validate  func(ctx context.Context, token *JSONWebToken, clientID string) (*Claims, error)
}

func (p *Provider) GetAuthURL(state string, redirectURL string) string {
	return fmt.Sprintf("%s?state=%s&client_id=%s&redirect_uri=%s&scope=%s&response_type=code&access_type=offline",
		p.AuthURL, state, p.ClientID, redirectURL, p.Scope)
}

func (p *Provider) ValidateToken(ctx context.Context, token *JSONWebToken) (*Claims, error) {
	return p.Validate(ctx, token, p.ClientID)
}

func (p *Provider) getToken(ctx context.Context, code string, redirectURI string) (*JSONWebToken, error) {
	resp, err := p.postRequest(ctx, p.TokenURL, url.Values{ //nolint:bodyclose
		"code":          {code},
		"client_id":     {p.ClientID},
		"client_secret": {p.Secret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, common.NewUnknownError(err, "unable-to-execute-code-exchange-request")
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msgf("Failed to close response body")
		}
	}(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		// TODO error struct
		content, cErr := io.ReadAll(resp.Body)
		if cErr != nil {
			return nil, common.NewUnknownError(cErr, "unable-to-read-code-exchange-error-response")
		}

		return nil, common.NewUnknownError(fmt.Errorf("code exchange endpoint error response %s", content),
			"code-exchange-endpoint-respond-with-error")
	}

	var jwtToken JSONWebToken
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jwtToken)
	if err != nil {
		return nil, common.NewUnknownError(err, "unable-to-decode-code-exchange-response")
	}

	jwtToken.Provider = p.Name

	return &jwtToken, nil
}

func (p *Provider) ExchangeCode(ctx context.Context, authCode string, redirectURI string) (*Claims,
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

func (p *Provider) RevokeToken(ctx context.Context, token string) error {
	resp, err := p.postRequest(ctx, p.RevokeURL, url.Values{ //nolint:bodyclose
		"token": {token},
	})
	if err != nil {
		return common.NewUnknownError(err, "unable-to-execute-token-revoke-request")
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msgf("Failed to close response body")
		}
	}(resp.Body.Close)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	content, cErr := io.ReadAll(resp.Body)
	if cErr != nil {
		return common.NewUnknownError(cErr, "unable-to-read-token-revoke-error-response")
	}

	return common.NewUnknownError(fmt.Errorf("token revoke endpoint error response %s", content),
		"revoke-token-endpoint-respond-with-error")
}

func (p *Provider) postRequest(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request, %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return p.Client.Do(req)
}
