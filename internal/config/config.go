package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database Database `yaml:"database"`
	Logging  Logging  `yaml:"logging"`
	Images   Images   `yaml:"images"`
	Server   Server   `yaml:"server"`
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

type Images struct {
	Host string `yaml:"host"`
}

type Server struct {
	Host        string `yaml:"host"`
	Cookie      Cookie `yaml:"cookie"`
	TemplateDir string `yaml:"template_path"`
	Port        int    `yaml:"port"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Cookie struct {
	// EncryptionKey a 32 character string
	EncryptionKey string `yaml:"encryption_key"`
}

type Oidc struct {
	Provider          map[string]Provider `yaml:"provider"`
	RedirectURI       string              `yaml:"redirect_uri"`
	SessionCookieName string              `yaml:"session_cookie_name"`
	StateCookieAge    time.Duration       `yaml:"state_cookie_age"`
	ClientTimeout     time.Duration       `yaml:"client_timeout"`
}

type Provider struct {
	AuthURL   string `yaml:"auth_url"`
	TokenURL  string `yaml:"token_url"`
	RevokeURL string `yaml:"revoke_url"`
	ClientID  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
	Scope     string `yaml:"scope"`
}

func NewConfig(path string) (*Config, error) {
	p := filepath.Clean(path)

	s, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read file info for %s, %w", p, err)
	}
	if s.IsDir() {
		return nil, fmt.Errorf("'%s' is a directory, not a regular file", p)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("can't read config file: %w", err)
	}

	defaultTimeoutSec := 5
	defaultConfig := Config{
		Logging: Logging{
			Level: "info",
		},
		Server: Server{
			TemplateDir: "./views",
		},
		Oidc: Oidc{
			SessionCookieName: "SESSION",
			StateCookieAge:    time.Minute,
			ClientTimeout:     time.Duration(defaultTimeoutSec) * time.Second,
		},
	}

	err = yaml.Unmarshal(data, &defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal failed with: %w", err)
	}

	// TODO: validate config content

	if strings.HasSuffix(defaultConfig.Images.Host, "") {
		defaultConfig.Images.Host += "/"
	}

	return &defaultConfig, nil
}
