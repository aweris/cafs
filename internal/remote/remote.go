// Package remote implements OCI registry operations.
//
// Based on go-containerregistry patterns:
// - Authentication via keychain
// - Upload ordering: layers → config → manifest
// - Standard OCI distribution spec
package remote

import "context"

// Remote handles OCI registry operations.
type Remote interface {
	// Push uploads a snapshot to the registry.
	Push(ctx context.Context, rootHash string, objects map[string][]byte) error

	// Pull downloads a snapshot from the registry.
	Pull(ctx context.Context) (rootHash string, objects map[string][]byte, err error)

	// GetRef retrieves the current hash for a reference.
	GetRef(ctx context.Context) (hash string, err error)
}
