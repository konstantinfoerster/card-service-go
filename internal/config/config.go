package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"gopkg.in/yaml.v3"
)

var (
	ErrReadFile       = errors.New("cannot read file")
	ErrInvalidContent = errors.New("unexpected file content")
)

type Config struct {
	Database postgres.Config `yaml:"database"`
	Logging  Logging         `yaml:"logging"`
	Images   postgres.Images `yaml:"images"`
	Server   web.Config      `yaml:"server"`
	Probes   web.Config      `yaml:"probes"`
	Auth     web.Auth        `yaml:"auth"`
}

type Logging struct {
	Level string `yaml:"level"`
}

func NewConfig(path string) (Config, error) {
	p := filepath.Clean(path)

	data, err := os.ReadFile(p)
	if err != nil {
		return Config{}, errors.Join(err, ErrReadFile)
	}

	defaultConfig := Config{
		Logging: Logging{
			Level: "info",
		},
		Server: web.Config{
			Port: 3000,
		},
		Probes: web.Config{
			Port: 3001,
		},
		Auth: web.Auth{
			HeaderUserID:    web.HeaderUserID,
			HeaderUserEmail: web.HeaderUserEmail,
			TestMode:        false,
			LoginURL:        "/login",
			LogoutURL:       "/logout",
		},
	}

	err = yaml.Unmarshal(data, &defaultConfig)
	if err != nil {
		return Config{}, errors.Join(err, ErrInvalidContent)
	}

	// TODO: validate config content

	if strings.HasSuffix(defaultConfig.Images.Host, "") {
		defaultConfig.Images.Host += "/"
	}

	return defaultConfig, nil
}
