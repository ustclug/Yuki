package auth

type NopAuthenticator struct {
}

func (c *NopAuthenticator) Authenticate(name, passwd string) error {
	return nil
}

func (c *NopAuthenticator) Cleanup() {
}

func NewNopAuthenticator() *NopAuthenticator {
	return &NopAuthenticator{}
}
