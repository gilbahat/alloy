//go:build linux

package vsock

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/mdlayher/vsock"
)

const scheme = "vsock://"

// Listen creates a net.Listener on the given vsock address.
//
// The address must be in the form "vsock://[CID]:PORT" or "vsock://:PORT".
// An empty or omitted CID listens on all context IDs (VMADDR_CID_ANY),
// equivalent to 0.0.0.0 for TCP.
func Listen(addr string) (net.Listener, error) {
	cid, port, err := parseAddr(addr)
	if err != nil {
		return nil, err
	}
	if cid == vmaddrCIDAny {
		return vsock.Listen(port, nil)
	}
	return vsock.ListenContextID(cid, port, nil)
}

// IsVsockAddress reports whether addr uses the vsock:// scheme.
func IsVsockAddress(addr string) bool {
	return strings.HasPrefix(addr, scheme)
}

// vmaddrCIDAny is VMADDR_CID_ANY, the vsock equivalent of 0.0.0.0.
const vmaddrCIDAny = ^uint32(0)

func parseAddr(addr string) (cid, port uint32, err error) {
	if !strings.HasPrefix(addr, scheme) {
		return 0, 0, fmt.Errorf("vsock address must start with %q, got %q", scheme, addr)
	}
	hostPort := strings.TrimPrefix(addr, scheme)
	idx := strings.LastIndex(hostPort, ":")
	if idx < 0 {
		return 0, 0, fmt.Errorf("vsock address %q must specify a port: use vsock://:PORT or vsock://CID:PORT", addr)
	}
	cidStr, portStr := hostPort[:idx], hostPort[idx+1:]

	p, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil || p == 0 {
		return 0, 0, fmt.Errorf("vsock address %q: invalid port %q (must be a non-zero uint32)", addr, portStr)
	}

	if cidStr == "" || cidStr == "*" {
		return vmaddrCIDAny, uint32(p), nil
	}
	c, err := strconv.ParseUint(cidStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("vsock address %q: invalid CID %q (must be a uint32 or empty for any)", addr, cidStr)
	}
	return uint32(c), uint32(p), nil
}
