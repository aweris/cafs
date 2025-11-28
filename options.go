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
	Registry  string
	CacheDir  string
	CacheSize int
	Auth      Authenticator
	AutoPull  string
}

// Option is a functional option for configuring CAS.
type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		CacheDir:  defaultCacheDir(),
		CacheSize: 1000,
		AutoPull:  "never",
	}
}

// WithRegistry sets the OCI registry URL.
func WithRegistry(url string) Option {
	return func(o *Options) { o.Registry = url }
}

// WithCacheDir sets the local cache directory.
func WithCacheDir(dir string) Option {
	return func(o *Options) { o.CacheDir = dir }
}

// WithCacheSize sets the in-memory cache size.
func WithCacheSize(size int) Option {
	return func(o *Options) { o.CacheSize = size }
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
