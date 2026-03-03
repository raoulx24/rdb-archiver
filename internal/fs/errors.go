package fs

import (
	"errors"
	"syscall"
)

// isTransient reports whether err represents a temporary filesystem condition.
// These cases are common on networked or RWM storage where brief contention,
// I/O stalls, or resource pressure can resolve on retry.
func isTransient(err error) bool {
	switch {
	// EAGAIN — Resource temporarily unavailable.
	// Often returned when the FS is momentarily busy or a lock cannot be acquired.
	case errors.Is(err, syscall.EAGAIN):

	// EBUSY — Device or resource busy.
	// Common when files are locked or the underlying storage is under contention.
	case errors.Is(err, syscall.EBUSY):

	// ETIMEDOUT — Operation timed out.
	// Typical for network filesystems experiencing latency spikes.
	case errors.Is(err, syscall.ETIMEDOUT):

	// ENFILE — System-wide file table overflow.
	// Usually transient on systems under temporary load.
	case errors.Is(err, syscall.ENFILE):

	// EMFILE — Process file descriptor limit reached.
	// Can be transient if descriptors are freed shortly after.
	case errors.Is(err, syscall.EMFILE):

	// EIO — I/O error.
	// On local disks this may be serious, but on network/RWM storage
	// it often indicates a temporary transport or backend issue.
	case errors.Is(err, syscall.EIO):

		return true
	}

	return false
}
