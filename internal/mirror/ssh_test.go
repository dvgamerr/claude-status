package mirror

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

func TestSSHInvokesExpectedCommandAndReportsBoundedFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()
	var captured []string
	mode := "success"
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		captured = append([]string{name}, args...)
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestSSHHelperProcess", "--", mode)
	}
	snapshot := model.Snapshot{SchemaVersion: model.CurrentSchemaVersion, Session: model.Session{ID: "test"}}
	if err := SSH(context.Background(), "pi@example:22", "/opt/status", snapshot); err != nil {
		t.Fatalf("SSH() success error = %v", err)
	}
	want := []string{"ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=3", "pi@example:22", "/opt/status", "import"}
	if strings.Join(captured, "|") != strings.Join(want, "|") {
		t.Fatalf("command args = %q, want %q", captured, want)
	}

	mode = "fail"
	if err := SSH(context.Background(), "pilab", DefaultRemoteBinary, snapshot); err == nil || !strings.Contains(err.Error(), "helper failure") {
		t.Fatalf("SSH() failure error = %v", err)
	}
}

func TestSSHHelperProcess(_ *testing.T) {
	if len(os.Args) < 2 || os.Args[len(os.Args)-2] != "--" {
		return
	}
	if os.Args[len(os.Args)-1] == "fail" {
		fmt.Fprintln(os.Stderr, "helper failure")
		os.Exit(9)
	}
	os.Exit(0)
}
