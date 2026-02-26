package postgres

import (
	"fmt"
	"net"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"` // #nosec G117 field is only used internaly
	MaxConns int32  `yaml:"max_conns"`
}

func (d Config) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s", d.Username, d.Password, net.JoinHostPort(d.Host, d.Port), d.Database)
}

type Images struct {
	Host string `yaml:"host"`
}
