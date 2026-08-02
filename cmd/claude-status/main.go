package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dvgamerr/claude-status/internal/app"
	"github.com/dvgamerr/claude-status/internal/service"
)

func main() {
	// Checked before anything else: the Windows Service Control Manager
	// expects a service process to call StartServiceCtrlDispatcher almost
	// immediately, so this can't wait for flag parsing or any other CLI
	// setup. IsWindowsService is always false outside Windows.
	if service.IsWindowsService() {
		err := service.RunAsService(app.RelayServiceName, func(ctx context.Context) error {
			if code := app.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); code != 0 {
				return fmt.Errorf("claude-status exited with code %d", code)
			}
			return nil
		})
		if err != nil {
			os.Exit(1)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
