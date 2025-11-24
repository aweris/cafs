package remote

// Authenticator provides authentication for OCI registry operations.
type Authenticator interface {
	// Authenticate returns credentials for the given registry.
	Authenticate(registry string) (username, password string, err error)
}

// DefaultAuthenticator uses the system keychain (like Docker).
type DefaultAuthenticator struct{}

// NewDefaultAuthenticator creates a default authenticator.
func NewDefaultAuthenticator() *DefaultAuthenticator {
	return &DefaultAuthenticator{}
}

// Authenticate returns credentials from the keychain.
func (a *DefaultAuthenticator) Authenticate(registry string) (string, string, error) {
	// TODO: Use go-containerregistry's authn.DefaultKeychain
	return "", "", nil
}
