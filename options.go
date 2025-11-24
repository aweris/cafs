package cafs

import (
	"os"
	"path/filepath"
)

// Options configures a CAFS workspace.
type Options struct {
	// Registry is the OCI registry URL
	// Default: registry.io (or from config)
	Registry string

	// CacheDir is the local cache directory
	// Default: ~/.cafs
	CacheDir string

	// CacheSize is the in-memory LRU cache size
	// Default: 1000 objects
	CacheSize int

	// ReadOnly opens the workspace in read-only mode
	// No local modifications allowed
	ReadOnly bool

	// Prefetch is a list of paths to eagerly load
	// Useful for known hot paths
	Prefetch []string

	// Auth provides custom authentication
	// Default: uses keychain (like Docker)
	Auth Authenticator

	// CompressionEnabled enables zstd compression for storage
	// Default: true
	CompressionEnabled bool

	// CompressionLevel sets the compression level (1=fast, 2=default, 3=best)
	// Default: 2 (balanced speed/ratio)
	CompressionLevel int

	// AutoPull controls automatic pulling from remote on Open
	// "never" (default): only load local cache
	// "missing": pull if local ref doesn't exist
	// "always": always pull latest from remote
	AutoPull string
}

// Option is a functional option for configuring CAFS.
type Option func(*Options)

// defaultOptions returns the default options.
func defaultOptions() *Options {
	return &Options{
		Registry:           "registry.io",
		CacheDir:           defaultCacheDir(),
		CacheSize:          1000,
		ReadOnly:           false,
		Prefetch:           nil,
		Auth:               nil,
		CompressionEnabled: true,
		CompressionLevel:   2,
		AutoPull:           "never",
	}
}

// WithRegistry sets the OCI registry URL.
func WithRegistry(url string) Option {
	return func(o *Options) {
		o.Registry = url
	}
}

// WithCacheDir sets the local cache directory.
func WithCacheDir(dir string) Option {
	return func(o *Options) {
		o.CacheDir = dir
	}
}

// WithCacheSize sets the in-memory cache size.
func WithCacheSize(size int) Option {
	return func(o *Options) {
		o.CacheSize = size
	}
}

// WithReadOnly opens the workspace in read-only mode.
func WithReadOnly() Option {
	return func(o *Options) {
		o.ReadOnly = true
	}
}

// WithPrefetch sets paths to eagerly load.
func WithPrefetch(paths []string) Option {
	return func(o *Options) {
		o.Prefetch = paths
	}
}

// WithAuth sets custom authentication.
func WithAuth(auth Authenticator) Option {
	return func(o *Options) {
		o.Auth = auth
	}
}

// WithOptions applies a full Options struct.
func WithOptions(opts *Options) Option {
	return func(o *Options) {
		*o = *opts
	}
}

// WithCompression enables or disables compression.
func WithCompression(enabled bool) Option {
	return func(o *Options) {
		o.CompressionEnabled = enabled
	}
}

// WithCompressionLevel sets the compression level (1=fast, 2=default, 3=best).
func WithCompressionLevel(level int) Option {
	return func(o *Options) {
		o.CompressionLevel = level
	}
}

// WithAutoPull enables automatic pulling from remote on Open.
// Use "missing" to pull only if local ref doesn't exist, or "always" to always sync.
func WithAutoPull(mode string) Option {
	return func(o *Options) {
		o.AutoPull = mode
	}
}

// WithAutoPullIfMissing pulls from remote on Open if local ref doesn't exist.
func WithAutoPullIfMissing() Option {
	return func(o *Options) {
		o.AutoPull = "missing"
	}
}

// WithAlwaysSync always pulls latest from remote on Open.
func WithAlwaysSync() Option {
	return func(o *Options) {
		o.AutoPull = "always"
	}
}

// defaultCacheDir returns the default cache directory.
// Uses XDG Base Directory specification.
func defaultCacheDir() string {
	// Try XDG_DATA_HOME first
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "cafs")
	}

	// Fall back to ~/.local/share/cafs
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cafs" // Last resort: current directory
	}

	return filepath.Join(home, ".local", "share", "cafs")
}
