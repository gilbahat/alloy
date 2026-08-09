package otelcol_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confignet"
)

// Convert passes timeouts through unchanged; defaults are applied by each
// component's SetToDefault, not here. That lets a component explicitly set
// a timeout to 0 (unbounded), which would otherwise be indistinguishable
// from "not set".
func TestHTTPServerArguments_ConvertTimeoutZeroValue(t *testing.T) {
	args := &otelcol.HTTPServerArguments{}
	cfg, err := args.Convert()
	require.NoError(t, err)

	server := cfg.Get()
	require.NotNil(t, server)
	require.Equal(t, time.Duration(0), server.IdleTimeout)
	require.Equal(t, time.Duration(0), server.ReadHeaderTimeout)
	require.Equal(t, time.Duration(0), server.WriteTimeout)
	require.Equal(t, time.Duration(0), server.ReadTimeout)
}

func TestHTTPServerArguments_ConvertTimeoutCustom(t *testing.T) {
	args := &otelcol.HTTPServerArguments{
		IdleTimeout:       2 * time.Minute,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      45 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}
	cfg, err := args.Convert()
	require.NoError(t, err)

	server := cfg.Get()
	require.NotNil(t, server)
	require.Equal(t, 2*time.Minute, server.IdleTimeout)
	require.Equal(t, 10*time.Second, server.ReadTimeout)
	require.Equal(t, 45*time.Second, server.WriteTimeout)
	require.Equal(t, 15*time.Second, server.ReadHeaderTimeout)
}

func TestHTTPServerArguments_ConvertTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		expect    confignet.TransportType
		expectErr string
	}{
		{
			name:      "empty defaults to tcp",
			transport: "",
			expect:    confignet.TransportTypeTCP,
		},
		{
			name:      "explicit tcp",
			transport: "tcp",
			expect:    confignet.TransportTypeTCP,
		},
		{
			name:      "tcp4",
			transport: "tcp4",
			expect:    confignet.TransportTypeTCP4,
		},
		{
			name:      "unix",
			transport: "unix",
			expect:    confignet.TransportTypeUnix,
		},
		{
			name:      "vsock",
			transport: "vsock",
			expect:    confignet.TransportTypeVsock,
		},
		{
			// http.Server needs a stream listener, so datagram transports are
			// rejected even though confignet itself accepts them.
			name:      "udp is rejected",
			transport: "udp",
			expectErr: `invalid transport "udp" for an HTTP server`,
		},
		{
			name:      "unknown is rejected",
			transport: "blarg",
			expectErr: `invalid transport "blarg" for an HTTP server`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &otelcol.HTTPServerArguments{
				Endpoint:  "0.0.0.0:4318",
				Transport: tt.transport,
			}
			cfg, err := args.Convert()
			if tt.expectErr != "" {
				require.ErrorContains(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)

			server := cfg.Get()
			require.NotNil(t, server)
			require.Equal(t, tt.expect, server.NetAddr.Transport)
			require.Equal(t, "0.0.0.0:4318", server.NetAddr.Endpoint)
		})
	}
}

// A vsock HTTP server takes its endpoint in confignet's "CID:PORT" form, which
// differs from the "vsock://CID:PORT" form the Alloy HTTP service and
// loki.source.syslog use.
func TestHTTPServerArguments_ConvertVsockEndpoint(t *testing.T) {
	args := &otelcol.HTTPServerArguments{
		Endpoint:  "3:4318",
		Transport: "vsock",
	}
	cfg, err := args.Convert()
	require.NoError(t, err)

	server := cfg.Get()
	require.NotNil(t, server)
	require.NoError(t, server.NetAddr.Validate())
}

// Serving over a non-TCP transport end-to-end. vsock needs a hypervisor, so
// this uses a unix socket, which reaches the same confignet listener path the
// vsock transport does. It guards against Convert regressing to a hardcoded
// TCP transport, which would make this listen on a TCP port named after the
// socket path instead.
func TestHTTPServerArguments_ServesOverNonTCPTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not available on Windows")
	}

	// Not t.TempDir(): its path blows past the ~104 byte sun_path limit on macOS.
	dir, err := os.MkdirTemp("/tmp", "alloy")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sock := filepath.Join(dir, "otlp.sock")
	args := &otelcol.HTTPServerArguments{
		Endpoint:  sock,
		Transport: "unix",
	}
	cfg, err := args.Convert()
	require.NoError(t, err)

	server := cfg.Get()
	require.NotNil(t, server)
	require.Equal(t, confignet.TransportTypeUnix, server.NetAddr.Transport)

	ln, err := server.ToListener(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	httpSrv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	t.Cleanup(func() { _ = httpSrv.Close() })
	go func() { _ = httpSrv.Serve(ln) }()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		},
	}

	req, err := http.NewRequest(http.MethodPost, "http://alloy/v1/traces", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "ok", string(body))
}
