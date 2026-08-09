//go:build !linux

package vsock

import (
	"fmt"
	"net"
	"strings"
)

const scheme = "vsock://"

// Listen always returns an error on non-Linux platforms.
func Listen(_ string) (net.Listener, error) {
	return nil, fmt.Errorf("vsock listeners are only supported on Linux")
}

// IsVsockAddress reports whether addr uses the vsock:// scheme.
func IsVsockAddress(addr string) bool {
	return strings.HasPrefix(addr, scheme)
}
