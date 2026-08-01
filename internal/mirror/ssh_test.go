package mirror

import (
	"context"
	"strings"
	"testing"

	"github.com/dvgamerr/claude-status/internal/model"
)

func TestSSHRejectsUnsafeTargetsBeforeLaunching(t *testing.T) {
	tests := []struct {
		name string
		host string
		bin  string
	}{
		{name: "empty host", bin: DefaultRemoteBinary},
		{name: "option host", host: "-oProxyCommand=bad", bin: DefaultRemoteBinary},
		{name: "shell host", host: "pi;bad", bin: DefaultRemoteBinary},
		{name: "shell binary", host: "pilab", bin: "/tmp/a;bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SSH(context.Background(), tt.host, tt.bin, model.Snapshot{})
			if err == nil || (!strings.Contains(err.Error(), "host") && !strings.Contains(err.Error(), "binary")) {
				t.Fatalf("SSH() error = %v", err)
			}
		})
	}
}
