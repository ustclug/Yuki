package auth

import (
	"errors"
)

type BasicAuthenticator struct {
	users map[string]string
}

var errBasicNotFound = errors.New("incorrect username or password")

func (c *BasicAuthenticator) Authenticate(name, passwd string) error {
	if passwd == c.users[name] {
		return nil
	}
	return errBasicNotFound
}

func (c *BasicAuthenticator) Cleanup() {
}

func NewBasicAuthenticator(users map[string]string) *BasicAuthenticator {
	return &BasicAuthenticator{users}
}
