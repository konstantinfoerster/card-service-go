package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrReadFile       = errors.New("cannot read file")
	ErrInvalidContent = errors.New("unexpected file content")
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
	MaxConns int32  `yaml:"max_conns"`
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
	SessionCookieName string              `yaml:"session_cookie_name"`
	StateCookieAge    time.Duration       `yaml:"state_cookie_age"`
	ClientTimeout     time.Duration       `yaml:"client_timeout"`
}

type Provider struct {
	AuthURL     string `yaml:"auth_url"`
	TokenURL    string `yaml:"token_url"`
	RevokeURL   string `yaml:"revoke_url"`
	RedirectURI string `yaml:"redirect_uri"`
	ClientID    string `yaml:"client_id"`
	Secret      string `yaml:"secret"`
	Scope       string `yaml:"scope"`
}

func NewConfig(path string) (*Config, error) {
	p := filepath.Clean(path)

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, errors.Join(err, ErrReadFile)
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
		return nil, errors.Join(err, ErrInvalidContent)
	}

	// TODO: validate config content

	if strings.HasSuffix(defaultConfig.Images.Host, "") {
		defaultConfig.Images.Host += "/"
	}

	return &defaultConfig, nil
}
