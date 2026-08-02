//go:build windows

package service

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Install registers cfg as a Windows Service (creating it if new, updating
// its binary path/description in place if it already exists) and starts it.
// StartAutomatic means it comes back on its own after a reboot, matching
// the always-on relay behavior the Scheduled Task approach used to give.
func Install(cfg Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()

	binaryPath := serviceBinaryPath(exePath, cfg.Args)
	s, err := m.OpenService(cfg.Name)
	if err == nil {
		defer s.Close()
		update := mgr.Config{
			ServiceType:    windows.SERVICE_NO_CHANGE,
			StartType:      mgr.StartAutomatic,
			ErrorControl:   windows.SERVICE_NO_CHANGE,
			DisplayName:    cfg.DisplayName,
			Description:    cfg.Description,
			BinaryPathName: binaryPath,
		}
		if err := s.UpdateConfig(update); err != nil {
			return fmt.Errorf("update service config: %w", err)
		}
		// UpdateConfig only changes what SCM will launch next time — a
		// currently running instance keeps executing the old binary path
		// in memory. Restart so a re-install (e.g. after upgrading the
		// binary) actually picks up the change instead of silently doing
		// nothing until the next reboot.
		if status, err := s.Query(); err == nil && status.State != svc.Stopped {
			if _, err := s.Control(svc.Stop); err != nil {
				return fmt.Errorf("stop running service before restart: %w", err)
			}
			if err := waitForState(s, svc.Stopped, 20*time.Second); err != nil {
				return fmt.Errorf("wait for service to stop before restart: %w", err)
			}
		}
	} else {
		s, err = m.CreateService(cfg.Name, exePath, mgr.Config{
			DisplayName:  cfg.DisplayName,
			Description:  cfg.Description,
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ErrorIgnore,
		}, cfg.Args...)
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
		defer s.Close()
	}

	if err := doStart(s); err != nil {
		return err
	}
	return waitForState(s, svc.Running, 20*time.Second)
}

// Remove stops name if running and deletes its service registration.
func Remove(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q is not installed: %w", name, err)
	}
	defer s.Close()

	if status, err := s.Query(); err == nil && status.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("stop service before removal: %w", err)
		}
		if err := waitForState(s, svc.Stopped, 20*time.Second); err != nil {
			return fmt.Errorf("wait for service to stop: %w", err)
		}
	}
	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

func Start(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q is not installed: %w", name, err)
	}
	defer s.Close()
	if err := doStart(s); err != nil {
		return err
	}
	return waitForState(s, svc.Running, 20*time.Second)
}

func Stop(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q is not installed: %w", name, err)
	}
	defer s.Close()
	if _, err := s.Control(svc.Stop); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}
	return waitForState(s, svc.Stopped, 20*time.Second)
}

func Status(name string) (State, error) {
	m, err := mgr.Connect()
	if err != nil {
		return StateNotInstalled, fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return StateNotInstalled, nil
	}
	defer s.Close()
	status, err := s.Query()
	if err != nil {
		return StateNotInstalled, fmt.Errorf("query service status: %w", err)
	}
	if status.State == svc.Running {
		return StateRunning, nil
	}
	return StateStopped, nil
}

func doStart(s *mgr.Service) error {
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("query service status: %w", err)
	}
	if status.State == svc.Running {
		return nil
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	return nil
}

func waitForState(s *mgr.Service, target svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("query service status: %w", err)
		}
		if status.State == target {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for service state %d", target)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func serviceBinaryPath(exePath string, args []string) string {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, syscall.EscapeArg(exePath))
	for _, arg := range args {
		quoted = append(quoted, syscall.EscapeArg(arg))
	}
	return strings.Join(quoted, " ")
}
