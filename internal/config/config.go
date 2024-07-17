package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database postgres.Database `yaml:"database"`
	Logging  Logging           `yaml:"logging"`
	Images   Images            `yaml:"images"`
	Server   web.ServerConfig  `yaml:"server"`
	Oidc     auth.OidcConfig   `yaml:"oidc"`
}

type Logging struct {
	Level string `yaml:"level"`
}

type Images struct {
	Host string `yaml:"host"`
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

	defaultConfig := Config{
		Logging: Logging{
			Level: "info",
		},
		Server: web.ServerConfig{
			TemplateDir: "./views",
		},
		Oidc: auth.OidcConfig{
			SessionCookieName: "SESSION",
			StateCookieAge:    time.Minute,
			ClientTimeout:     5 * time.Second,
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
