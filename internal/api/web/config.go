package web

import "fmt"

type Mode string

const (
	dev  Mode = "dev"
	prod Mode = "prod"
)

// DefaultMaxBodySize request body limit in bytes (1 MiB).
const DefaultMaxBodySize = 1024 * 1024

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	TLS  TLS    `yaml:"tls"`
	Mode Mode   `yaml:"mode"`
	// MaxBodySize maximum accepted request body size in bytes,
	// defaults to 1 MiB if not set.
	MaxBodySize int `yaml:"max_body_size"`
	// Debug enables debug features like saving image posted that are received by the web-api.
	Debug bool `yaml:"debug"`
	// DebugDir directory where the debug output is written to,
	// defaults to "debug" if empty.
	DebugDir string `yaml:"debug_dir"`
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
