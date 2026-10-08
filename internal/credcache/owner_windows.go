//go:build windows

package credcache

import "os"

// Windows has no uid ownership model that applies here; ACLs on the user
// cache dir already restrict it to the user.
var ownedByUs = func(os.FileInfo) bool { return true }
