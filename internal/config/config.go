package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"go.yaml.in/yaml/v3"
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

func ReadConfigs(path ...string) (Config, error) {
	cfg := Config{
		Logging: Logging{
			Level: "info",
		},
		Server: web.Config{
			Port:     3000,
			Debug:    false,
			DebugDir: "debug",
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

	for _, p := range path {
		p = filepath.Clean(p)

		configRaw, err := os.ReadFile(p)
		if err != nil {
			return Config{}, errors.Join(err, ErrReadFile)
		}

		cfg, err = readConfig(configRaw, cfg)
		if err != nil {
			return Config{}, errors.Join(err, ErrInvalidContent)
		}
	}

	return cfg, nil
}

func readConfig(configRaw []byte, target Config) (Config, error) {
	err := yaml.Unmarshal(configRaw, &target)
	if err != nil {
		return Config{}, err
	}

	// TODO: validate config content

	if strings.HasSuffix(target.Images.Host, "") {
		target.Images.Host += "/"
	}

	return target, nil
}
