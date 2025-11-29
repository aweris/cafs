package cafs

import "errors"

var (
	ErrNotFound = errors.New("cafs: not found")
	ErrNoRemote = errors.New("cafs: no remote configured")
)
