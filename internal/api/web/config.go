package web

import "fmt"

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	TLS  TLS    `yaml:"tls"`
}

func (c Config) Addr() string {
	defaultPort := 3000
	if c.Port < 1 {
		c.Port = defaultPort
	}

	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type TLS struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	Enabled  bool   `yaml:"enabled"`
}

type Auth struct {
	LoginURL        string `yaml:"login_url"`
	LogoutURL       string `yaml:"logout_url"`
	HeaderUserID    string `yaml:"header_user_id"`
	HeaderUserEmail string `yaml:"header_user_email"`
	TestMode        bool   `yaml:"test_mode"`
	UserID          string `yaml:"user_id"`
	UserEmail       string `yaml:"user_email"`
}
