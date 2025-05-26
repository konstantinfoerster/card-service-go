package web

import "fmt"

type Config struct {
	Host        string `yaml:"host"`
	Cookie      Cookie `yaml:"cookie"`
	TemplateDir string `yaml:"template_path"`
	TLS         TLS    `yaml:"tls"`
	Port        int    `yaml:"port"`
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

type Cookie struct {
	// EncryptionKey a 32 character string
	EncryptionKey string `yaml:"encryption_key"`
}
