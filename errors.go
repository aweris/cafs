package cafs

import "errors"

var (
	ErrNotFound    = errors.New("cafs: not found")
	ErrNoRemote    = errors.New("cafs: no remote configured")
	ErrReservedKey = errors.New("cafs: key prefix '_' is reserved")
	ErrInvalidKey  = errors.New("cafs: invalid key")
)
