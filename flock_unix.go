//go:build unix

package dynconfig

import (
	"os"

	"golang.org/x/sys/unix"
)

// fsLockSupported reports whether exclusive OS-level file locking is
// implemented on this platform. It is true on Unix, where flock(2) is used.
const fsLockSupported = true

// lockFileExclusive acquires a blocking exclusive (write) advisory lock
// on the open file using flock(2). It blocks until the lock can be acquired.
//
// The lock is associated with the open file description and is released
// when the file is closed or unlockFile is called.
func lockFileExclusive(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

// unlockFile releases an advisory lock previously acquired with lockFileExclusive.
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
