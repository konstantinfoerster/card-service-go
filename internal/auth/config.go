package auth

import "time"

type Config struct {
	Provider          map[string]ProviderCfg `yaml:"provider"`
	SessionCookieName string                 `yaml:"session_cookie_name"`
	StateCookieAge    time.Duration          `yaml:"state_cookie_age"`
	ClientTimeout     time.Duration          `yaml:"client_timeout"`
}

type ProviderCfg struct {
	AuthURL     string `yaml:"auth_url"`
	TokenURL    string `yaml:"token_url"`
	RevokeURL   string `yaml:"revoke_url"`
	RedirectURI string `yaml:"redirect_uri"`
	ClientID    string `yaml:"client_id"`
	Secret      string `yaml:"secret"`
	Scope       string `yaml:"scope"`
}
