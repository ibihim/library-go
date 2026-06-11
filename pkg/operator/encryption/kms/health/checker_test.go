package health

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	kmsservice "k8s.io/kms/pkg/service"
)

type fakeService struct {
	resp *kmsservice.StatusResponse
	err  error
}

func (f *fakeService) Status(context.Context) (*kmsservice.StatusResponse, error) {
	return f.resp, f.err
}
func (f *fakeService) Encrypt(context.Context, string, []byte) (*kmsservice.EncryptResponse, error) {
	return nil, nil
}
func (f *fakeService) Decrypt(context.Context, string, *kmsservice.DecryptRequest) ([]byte, error) {
	return nil, nil
}

func TestChecker_CheckStatus(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	c := &checker{
		plugins: []plugin{
			{keyID: "1", service: &fakeService{resp: &kmsservice.StatusResponse{Healthz: "ok", KeyID: "kek-abc"}}},
			{keyID: "2", service: &fakeService{err: fmt.Errorf("connection refused")}},
			{keyID: "3", service: &fakeService{resp: &kmsservice.StatusResponse{Healthz: "degraded"}}},
		},
		now: func() time.Time { return fixed },
	}

	have := c.checkStatus(context.Background())
	want := []PluginHealthReport{
		{KeyID: "1", KEKID: "kek-abc", Status: "healthy", LastChecked: fixed},
		{KeyID: "2", Status: "error", Detail: "connection refused", LastChecked: fixed},
		{KeyID: "3", Status: "unhealthy", Detail: "degraded", LastChecked: fixed},
	}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("checkStatus():\n have: %+v\n want: %+v", have, want)
	}
}

func Test_keyIDFromSocket(t *testing.T) {
	tests := []struct {
		socket  string
		want    string
		wantErr bool
	}{
		{socket: "unix:///var/run/kmsplugin/kms-1.sock", want: "1"},
		{socket: "unix:///var/run/kmsplugin/kms-42.sock", want: "42"},
		{socket: "unix:///var/run/kmsplugin/plugin.sock", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.socket, func(t *testing.T) {
			have, err := keyIDFromSocket(tt.socket)
			if (err != nil) != tt.wantErr {
				t.Fatalf("keyIDFromSocket(%q) err = %v, wantErr %v", tt.socket, err, tt.wantErr)
			}
			if have != tt.want {
				t.Errorf("keyIDFromSocket(%q) = %q, want %q", tt.socket, have, tt.want)
			}
		})
	}
}
