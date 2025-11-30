package cafs

import (
	"os"
	"path/filepath"

	"github.com/aweris/cafs/internal/remote"
)

// AutoPull modes
const (
	AutoPullNever   = "never"
	AutoPullAlways  = "always"
	AutoPullMissing = "missing"
)

// Authenticator provides credentials for remote registries.
type Authenticator = remote.Authenticator

// OpenOptions configures a CAS store.
type OpenOptions struct {
	CacheDir    string
	Auth        Authenticator
	AutoPull    string
	Concurrency int
}

// OpenOption is a functional option for configuring Open.
type OpenOption func(*OpenOptions)

func defaultOptions() *OpenOptions {
	return &OpenOptions{
		CacheDir:    defaultCacheDir(),
		AutoPull:    AutoPullNever,
		Concurrency: remote.DefaultConcurrency,
	}
}

// WithCacheDir sets the local cache directory.
func WithCacheDir(dir string) OpenOption {
	return func(o *OpenOptions) { o.CacheDir = dir }
}

// WithAuth sets custom authentication.
func WithAuth(auth Authenticator) OpenOption {
	return func(o *OpenOptions) { o.Auth = auth }
}

// WithAutoPull enables automatic pulling from remote on Open.
func WithAutoPull(mode string) OpenOption {
	return func(o *OpenOptions) { o.AutoPull = mode }
}

// WithConcurrency sets the number of parallel operations for push/pull.
func WithConcurrency(n int) OpenOption {
	return func(o *OpenOptions) {
		if n > 0 {
			o.Concurrency = n
		}
	}
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
