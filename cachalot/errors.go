package cachalot

import "errors"

var (
	// ErrContainerNotFound is returned when a container is not present in the cache.
	ErrContainerNotFound = errors.New("container not found")
	// ErrImageNotFound is returned when an image is not present in the cache.
	ErrImageNotFound = errors.New("image not found")
)
