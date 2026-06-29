//go:build !unix

package dynconfig

import "os"

// fsLockSupported reports whether exclusive OS-level file locking is
// implemented on this platform. It is false here because no portable
// implementation exists for this GOOS (flock(2) is Unix-only, and Windows
// LockFileEx has different, mandatory-locking semantics).
//
// When false, Set falls back to an in-place overwrite without an OS lock or
// atomic rename, protected only by the Loader's in-process mutex (so it is not
// safe against other processes mutating the same file).
const fsLockSupported = false

// lockFileExclusive is a no-op on platforms without file locking support.
// It is never called because Set guards the lock with fsLockSupported.
func lockFileExclusive(*os.File) error {
	return nil
}

// unlockFile is a no-op on platforms without file locking support.
func unlockFile(*os.File) error {
	return nil
}
