//go:build !windows

package service

import "testing"

func TestIsWindowsServiceAlwaysFalse(t *testing.T) {
	if IsWindowsService() {
		t.Fatal("IsWindowsService() = true on a non-Windows build")
	}
}

func TestRunAsServiceUnsupported(t *testing.T) {
	if err := RunAsService("any", nil); err != ErrUnsupportedPlatform {
		t.Fatalf("RunAsService() error = %v, want %v", err, ErrUnsupportedPlatform)
	}
}
