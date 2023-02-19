package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`
	Server   Server   `yaml:"server"`
	Images   Images   `yaml:"images"`
	Oidc     Oidc     `yaml:"oidc"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (d Database) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s", d.Username, d.Password, net.JoinHostPort(d.Host, d.Port), d.Database)
}

type Logging struct {
	Level string `yaml:"level"`
}

func (l Logging) LevelOrDefault() string {
	level := strings.TrimSpace(l.Level)
	if level == "" {
		level = "INFO"
	}

	return strings.ToLower(level)
}

type Server struct {
	Port   int    `yaml:"port"`
	Cookie Cookie `yaml:"cookie"`
}

func (s Server) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

type Cookie struct {
	EncryptionKey string `yaml:"encryption_key"` // must be a 32 character string
}

type Oidc struct {
	RedirectURI       string              `yaml:"redirect_uri"`
	SessionCookieName string              `yaml:"session_cookie_name"`
	Provider          map[string]Provider `yaml:"provider"`
}

type Provider struct {
	AuthURL   string `yaml:"auth_url"`
	TokenURL  string `yaml:"token_url"`
	RevokeURL string `yaml:"revoke_url"`
	ClientID  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
	Scope     string `yaml:"scope"`
}

type Images struct {
	Host string `yaml:"host"`
}

func NewConfig(path string) (*Config, error) {
	s, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file info for %s, %w", path, err)
	}
	if s.IsDir() {
		return nil, fmt.Errorf("'%s' is a directory, not a regular file", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read config file: %w", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal failed with: %w", err)
	}

	// TODO validate config content

	return config, nil
}
