package health

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	k8senvelopekmsv2 "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/kmsv2"
)

// validOptions returns an options value that passes validate. Each test case
// mutates a single field so the failure under test is unambiguous.
func validOptions() *options {
	return &options{
		KMSSockets:   []string{"unix:///var/run/kmsplugin/kms-1.sock"},
		Interval:     30 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		NodeName:     "node-1",
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*options)
		wantErr bool
	}{
		{
			name:   "valid",
			mutate: func(*options) {},
		},
		{
			name:    "no sockets",
			mutate:  func(o *options) { o.KMSSockets = nil },
			wantErr: true,
		},
		{
			name:    "empty socket entry",
			mutate:  func(o *options) { o.KMSSockets = []string{""} },
			wantErr: true,
		},
		{
			name:   "multiple valid sockets",
			mutate: func(o *options) { o.KMSSockets = append(o.KMSSockets, "unix:///var/run/kmsplugin/kms-2.sock") },
		},
		{
			name:    "socket missing unix scheme",
			mutate:  func(o *options) { o.KMSSockets = []string{"/var/run/kmsplugin/kms-1.sock"} },
			wantErr: true,
		},
		{
			name:    "socket scheme without path",
			mutate:  func(o *options) { o.KMSSockets = []string{"unix://"} },
			wantErr: true,
		},
		{
			name:    "socket wrong directory",
			mutate:  func(o *options) { o.KMSSockets = []string{"unix:///tmp/kms-1.sock"} },
			wantErr: true,
		},
		{
			name:    "socket non-numeric index",
			mutate:  func(o *options) { o.KMSSockets = []string{"unix:///var/run/kmsplugin/kms-x.sock"} },
			wantErr: true,
		},
		{
			name:    "socket missing .sock suffix",
			mutate:  func(o *options) { o.KMSSockets = []string{"unix:///var/run/kmsplugin/kms-1"} },
			wantErr: true,
		},
		{
			name:    "socket with surrounding whitespace",
			mutate:  func(o *options) { o.KMSSockets = []string{" unix:///var/run/kmsplugin/kms-1.sock "} },
			wantErr: true,
		},
		{
			name:    "interval zero",
			mutate:  func(o *options) { o.Interval = 0 },
			wantErr: true,
		},
		{
			name:    "interval negative",
			mutate:  func(o *options) { o.Interval = -time.Second },
			wantErr: true,
		},
		{
			name:    "read timeout zero",
			mutate:  func(o *options) { o.ReadTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "write timeout zero",
			mutate:  func(o *options) { o.WriteTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "node name empty",
			mutate:  func(o *options) { o.NodeName = "" },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := validOptions()
			tc.mutate(o)

			err := o.validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// buildPlugins fails fast on any NewGRPCService error, which is only safe if
// such an error can never mean "plugin down". This pins the vendored behavior:
// construction performs no I/O (a well-formed endpoint without a listening
// socket succeeds; reachability surfaces at the Status RPC instead), so the
// only error left is a malformed endpoint.
func TestNewGRPCServiceErrsOnlyOnParse(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "well-formed endpoint without a listening socket",
			endpoint: "unix://" + filepath.Join(t.TempDir(), "kms-1.sock"),
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  true,
		},
		{
			name:     "non-unix scheme",
			endpoint: "https://localhost:1234",
			wantErr:  true,
		},
		{
			name:     "missing scheme",
			endpoint: "/var/run/kmsplugin/kms-1.sock",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service, err := k8senvelopekmsv2.NewGRPCService(t.Context(), tc.endpoint, providerName, time.Second)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, service)
		})
	}
}
