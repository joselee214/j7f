package email

import (
	"crypto/tls"
	"gopkg.in/gomail.v2"
)

type Config struct {
	SmtpAddr   string
	Port       int
	User       string
	Password   string
	SenderName string
	TlsConfig  *tls.Config

	ToUsers []string
}

func New(c *Config) *gomail.Dialer {
	d := gomail.NewPlainDialer(c.SmtpAddr, c.Port, c.User, c.Password)
	if c.TlsConfig != nil {
		d.TLSConfig = c.TlsConfig
	}

	return d
}
