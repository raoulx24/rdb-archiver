package fs

import (
	"errors"
	"syscall"
)

// defines helpers for detecting transient filesystem errors.
// These determine whether an operation should retry or fail immediately.

func isTransient(err error) bool {
	if errors.Is(err, syscall.EAGAIN) ||
		errors.Is(err, syscall.EBUSY) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// extend here for cloud FS specific errors if needed
	return false
}
