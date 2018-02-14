package auth

type Authenticator interface {
	Authenticate(name, passwd string) error
	Cleanup()
}
