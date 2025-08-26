package auth

import "time"

type Config struct {
	Provider      map[string]ProviderCfg `yaml:"provider"`
	Session       Cookie                 `yaml:"session"`
	State         Cookie                 `yaml:"state"`
	ClientTimeout time.Duration          `yaml:"client_timeout"`
}

type Cookie struct {
	Name      string        `yaml:"name"`
	ExpiresIn time.Duration `yaml:"expires_in"`
	SameSite  string        `yaml:"same_site"`
	Domain    string        `yaml:"domain"`
	Path      string        `yaml:"path"`
}

type ProviderCfg struct {
	AuthURL     string   `yaml:"auth_url"`
	TokenURL    string   `yaml:"token_url"`
	RevokeURL   string   `yaml:"revoke_url"`
	RedirectURI string   `yaml:"redirect_uri"`
	ClientID    string   `yaml:"client_id"`
	Secret      string   `yaml:"secret"`
	Scopes      []string `yaml:"scopes"`
}
