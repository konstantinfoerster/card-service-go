package auth

import "time"

type OidcConfig struct {
	Provider          map[string]ProviderConfig `yaml:"provider"`
	RedirectURI       string                    `yaml:"redirect_uri"`
	SessionCookieName string                    `yaml:"session_cookie_name"`
	StateCookieAge    time.Duration             `yaml:"state_cookie_age"`
	ClientTimeout     time.Duration             `yaml:"client_timeout"`
}

type ProviderConfig struct {
	AuthURL   string `yaml:"auth_url"`
	TokenURL  string `yaml:"token_url"`
	RevokeURL string `yaml:"revoke_url"`
	ClientID  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
	Scope     string `yaml:"scope"`
}
