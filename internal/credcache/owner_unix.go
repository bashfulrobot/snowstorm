//go:build !windows

package credcache

import (
	"os"
	"syscall"
)

// ownedByUs reports whether fi is owned by the current uid. It is a variable
// so tests can simulate a foreign owner without needing root.
var ownedByUs = func(fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
