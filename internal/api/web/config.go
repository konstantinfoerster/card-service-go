package web

import "fmt"

type ServerConfig struct {
	Host        string `yaml:"host"`
	Cookie      Cookie `yaml:"cookie"`
	TemplateDir string `yaml:"template_path"`
	Port        int    `yaml:"port"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Cookie struct {
	// EncryptionKey a 32 character string
	EncryptionKey string `yaml:"encryption_key"`
}
