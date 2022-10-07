package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type Config struct {
	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`
	Server   Server   `yaml:"server"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (d Database) ConnectionUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", d.Username, d.Password, d.Host, d.Port, d.Database)
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
	Port int `yaml:"port"`
}

func (s Server) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

func NewConfig(path string) (*Config, error) {
	s, err := os.Stat(path)
	if err != nil {
		return nil, err
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
