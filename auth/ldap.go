package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	ldap "gopkg.in/ldap.v2"
)

type LdapAuthenticator struct {
	Conn    ldap.Client
	Config  *LdapAuthConfig
	RootCAs *x509.CertPool
}

type LdapAuthConfig struct {
	Attributes         []string `mapstructure:"attributes"`
	Base               string   `mapstructure:"base"`
	GroupFilter        string   `mapstructure:"group_filter"` // e.g. "(memberUid=%s)"
	Host               string   `mapstructure:"host"`
	UserFilter         string   `mapstructure:"user_filter"` // e.g. "(uid=%s)"
	Port               int      `mapstructure:"port"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
	UseSSL             bool     `mapstructure:"use_ssl"`
	CACertificates     []string `mapstructure:"ca_certificates"`
	//ClientCertificates []tls.Certificate // Adding client certificates
}

func (c *LdapAuthenticator) connect() error {
	var (
		l   *ldap.Conn
		err error
	)

	if c.Config.Port == 0 {
		if c.Config.UseSSL {
			c.Config.Port = 636
		} else {
			c.Config.Port = 389
		}
	}

	addr := fmt.Sprintf("%s:%d", c.Config.Host, c.Config.Port)
	if c.Config.UseSSL {
		tlsCfg := tls.Config{
			InsecureSkipVerify: c.Config.InsecureSkipVerify,
			ServerName:         c.Config.Host,
		}
		if len(c.Config.CACertificates) > 0 {
			tlsCfg.RootCAs = x509.NewCertPool()
			for _, cert := range c.Config.CACertificates {
				b, err := ioutil.ReadFile(cert)
				if err != nil {
					return err
				}
				if !tlsCfg.RootCAs.AppendCertsFromPEM(b) {
					return fmt.Errorf("could not append CA certs from PEM")
				}
			}
		}
		l, err = ldap.DialTLS("tcp", addr, &tlsCfg)
	} else {
		l, err = ldap.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	c.Conn = l
	return nil
}

func (c *LdapAuthenticator) Authenticate(name, passwd string) error {
	attributes := append(c.Config.Attributes, "dn")
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		c.Config.Base,
		// set timeLimit to 10 secs
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 10, false,
		fmt.Sprintf(c.Config.UserFilter, name),
		attributes,
		nil,
	)

	sr, err := c.Conn.Search(searchRequest)
	if err != nil {
		return err
	}

	if len(sr.Entries) < 1 {
		return fmt.Errorf("user does not exist")
	}

	if len(sr.Entries) > 1 {
		return fmt.Errorf("too many entries returned")
	}

	userDN := sr.Entries[0].DN
	err = c.Conn.Bind(userDN, passwd)
	return err
}

// Cleanup - close the backend ldap connection
func (c *LdapAuthenticator) Cleanup() {
	c.Conn.Close()
	c.Conn = nil
}

func NewLdapAuthenticator(cfg *LdapAuthConfig) (*LdapAuthenticator, error) {
	c := &LdapAuthenticator{}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid=%s)"
	}
	c.Config = cfg
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}
