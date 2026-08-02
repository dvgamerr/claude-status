//go:build windows

package service

import (
	"context"
	"time"

	"golang.org/x/sys/windows/svc"
)

// IsWindowsService reports whether this process was started by the Windows
// Service Control Manager rather than from an interactive session. main()
// checks this before doing anything else: SCM expects the process to call
// StartServiceCtrlDispatcher almost immediately, so this can't wait for
// flag parsing or any other CLI setup first.
func IsWindowsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		// Can't tell — assume a normal interactive/CLI run, the safer
		// default, rather than entering the SCM dispatch loop and hanging
		// a plain terminal invocation.
		return false
	}
	return isService
}

// RunAsService hands control to the Windows Service Control Manager: run is
// started in the background immediately, and a Stop/Shutdown control
// request cancels its context and waits (up to 10s) for it to return before
// reporting StopPending -> Stopped back to SCM.
func RunAsService(name string, run func(ctx context.Context) error) error {
	return svc.Run(name, &handler{run: run})
}

type handler struct {
	run func(ctx context.Context) error
}

func (h *handler) Execute(_ []string, requests <-chan svc.ChangeRequest, statusChan chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	statusChan <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.run(ctx) }()

	statusChan <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case <-done:
			cancel()
			statusChan <- svc.Status{State: svc.StopPending}
			return false, 0
		case req := <-requests:
			switch req.Cmd {
			case svc.Interrogate:
				statusChan <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				cancel()
				select {
				case <-done:
				case <-time.After(10 * time.Second):
				}
				statusChan <- svc.Status{State: svc.StopPending}
				return false, 0
			default:
				// Unexpected control requests are ignored, not fatal.
			}
		}
	}
}
