package cafs

import "errors"

// Common errors returned by CAFS operations.
var (
	// ErrNotImplemented indicates functionality not yet implemented.
	ErrNotImplemented = errors.New("cafs: not implemented")

	// ErrNotFound indicates an object was not found in storage.
	ErrNotFound = errors.New("cafs: not found")

	// ErrInvalidHash indicates a malformed content hash.
	ErrInvalidHash = errors.New("cafs: invalid hash")

	// ErrInvalidRef indicates a malformed namespace:ref string.
	ErrInvalidRef = errors.New("cafs: invalid reference format")

	// ErrDirtyWorkspace indicates the workspace has uncommitted changes.
	ErrDirtyWorkspace = errors.New("cafs: workspace has uncommitted changes")

	// ErrConflict indicates a conflict during Pull operation.
	ErrConflict = errors.New("cafs: conflict detected")

	// ErrReadOnly indicates a write operation on a read-only workspace.
	ErrReadOnly = errors.New("cafs: workspace is read-only")

	// ErrCorrupted indicates corrupted data (hash mismatch).
	ErrCorrupted = errors.New("cafs: corrupted data")
)
