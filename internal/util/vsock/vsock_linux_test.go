//go:build linux

package vsock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantCID uint32
		wantPort uint32
		wantErr string
	}{
		{
			name:     "any CID, explicit port",
			addr:     "vsock://:1234",
			wantCID:  vmaddrCIDAny,
			wantPort: 1234,
		},
		{
			name:     "wildcard CID",
			addr:     "vsock://*:1234",
			wantCID:  vmaddrCIDAny,
			wantPort: 1234,
		},
		{
			name:     "specific CID",
			addr:     "vsock://3:5000",
			wantCID:  3,
			wantPort: 5000,
		},
		{
			name:     "hypervisor CID",
			addr:     "vsock://1:9000",
			wantCID:  1,
			wantPort: 9000,
		},
		{
			name:    "missing scheme",
			addr:    "3:1234",
			wantErr: `vsock address must start with "vsock://"`,
		},
		{
			name:    "missing port",
			addr:    "vsock://3",
			wantErr: "must specify a port",
		},
		{
			name:    "invalid port",
			addr:    "vsock://:abc",
			wantErr: "invalid port",
		},
		{
			name:    "zero port",
			addr:    "vsock://:0",
			wantErr: "invalid port",
		},
		{
			name:    "invalid CID",
			addr:    "vsock://notacid:1234",
			wantErr: "invalid CID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid, port, err := parseAddr(tt.addr)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCID, cid)
			assert.Equal(t, tt.wantPort, port)
		})
	}
}

func Test_IsVsockAddress(t *testing.T) {
	assert.True(t, IsVsockAddress("vsock://:1234"))
	assert.True(t, IsVsockAddress("vsock://3:1234"))
	assert.False(t, IsVsockAddress("0.0.0.0:1234"))
	assert.False(t, IsVsockAddress("unix:///tmp/sock"))
	assert.False(t, IsVsockAddress(""))
}
