//go:build windows

package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/sys/windows/svc"
)

func TestHandlerReportsWorkerFailure(t *testing.T) {
	wantErr := errors.New("worker failed")
	h := &handler{run: func(context.Context) error { return wantErr }}
	requests := make(chan svc.ChangeRequest)
	statuses := make(chan svc.Status, 3)
	specific, code := h.Execute(nil, requests, statuses)
	if specific || code == 0 {
		t.Fatalf("Execute() = %v, %d; want non-specific failure", specific, code)
	}
}

func TestHandlerStopsOnRequest(t *testing.T) {
	h := &handler{run: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	requests := make(chan svc.ChangeRequest, 1)
	requests <- svc.ChangeRequest{Cmd: svc.Stop}
	statuses := make(chan svc.Status, 3)
	specific, code := h.Execute(nil, requests, statuses)
	if specific || code != 0 {
		t.Fatalf("Execute() = %v, %d; want clean stop", specific, code)
	}
}
