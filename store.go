package cafs

import (
	"context"

	"github.com/aweris/cafs/internal/store"
)

// Store is the public interface for content storage.
// Re-exported from internal/store for convenience.
type Store = store.Store

// Remote is the public interface for OCI registry operations.
// Re-exported from internal/remote for convenience.
type Remote interface {
	Push(ctx context.Context, rootHash string, objects map[string][]byte) error
	Pull(ctx context.Context) (rootHash string, objects map[string][]byte, err error)
	GetRef(ctx context.Context) (hash string, err error)
}

// Authenticator provides authentication for remote operations.
type Authenticator interface {
	Authenticate(registry string) (username, password string, err error)
}
