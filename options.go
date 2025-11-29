package cafs

import (
	"os"
	"path/filepath"

	"github.com/aweris/cafs/internal/remote"
)

// Authenticator provides credentials for remote registries.
type Authenticator = remote.Authenticator

// Options configures a CAS store.
type Options struct {
	CacheDir string
	Auth     Authenticator
	AutoPull string
}

// Option is a functional option for configuring CAS.
type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		CacheDir: defaultCacheDir(),
		AutoPull: "never",
	}
}

// WithCacheDir sets the local cache directory.
func WithCacheDir(dir string) Option {
	return func(o *Options) { o.CacheDir = dir }
}

// WithAuth sets custom authentication.
func WithAuth(auth Authenticator) Option {
	return func(o *Options) { o.Auth = auth }
}

// WithAutoPull enables automatic pulling from remote on Open.
func WithAutoPull(mode string) Option {
	return func(o *Options) { o.AutoPull = mode }
}

func defaultCacheDir() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "cafs")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "cafs")
	}
	return ".cafs"
}
