package health

// WaitForReady is a per-RPC option (stored by WithDefaultCallOptions, consumed
// at call time); it cannot make grpc.Dial wait for the remote side. The dial
// option with that behavior is WithBlock, which the kmsv2 code does not use.

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	kmsapi "k8s.io/kms/apis/v2"
)

func TestWaitForReadyActsOnRPCsNotOnDial(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "dead.sock") // nothing listens here, ever
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	dialer := grpc.WithContextDialer(func(_ context.Context, addr string) (net.Conn, error) {
		return net.DialUnix("unix", nil, &net.UnixAddr{Name: addr}) // same as kmsv2 grpc_service.go
	})
	waitForReady := grpc.WithDefaultCallOptions(grpc.WaitForReady(true))

	// 1: the exact k8s option set, dead socket
	start := time.Now()
	cc, err := grpc.Dial(sock, creds, dialer, waitForReady)
	t.Logf("1 Dial + WaitForReady(true), dead socket:  err=%v  elapsed=%s", err, time.Since(start).Round(time.Millisecond))
	require.NoError(t, err)

	client := kmsapi.NewKeyManagementServiceClient(cc)

	// 2: where WaitForReady actually acts: the RPC waits for the deadline
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	start = time.Now()
	_, err = client.Status(ctx, &kmsapi.StatusRequest{})
	t.Logf("2 Status RPC, WaitForReady(true):          err=%v  elapsed=%s", err, time.Since(start).Round(time.Millisecond))
	require.Error(t, err)

	// 3: flip it off per-call: fail fast, no waiting
	start = time.Now()
	_, err = client.Status(t.Context(), &kmsapi.StatusRequest{}, grpc.WaitForReady(false))
	t.Logf("3 Status RPC, WaitForReady(false):         err=%v  elapsed=%s", err, time.Since(start).Round(time.Millisecond))
	require.Error(t, err)

	// 4: the option that DOES make Dial wait for the other party: WithBlock
	ctx2, cancel2 := context.WithTimeout(t.Context(), time.Second)
	defer cancel2()
	start = time.Now()
	_, err = grpc.DialContext(ctx2, sock, creds, dialer, grpc.WithBlock())
	t.Logf("4 DialContext + WithBlock, dead socket:    err=%v  elapsed=%s", err, time.Since(start).Round(time.Millisecond))
	require.Error(t, err)
}
