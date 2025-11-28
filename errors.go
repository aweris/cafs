package cafs

import "errors"

var (
	ErrNotFound   = errors.New("cafs: not found")
	ErrInvalidRef = errors.New("cafs: invalid reference format")
	ErrNoRemote   = errors.New("cafs: no remote configured")
)
