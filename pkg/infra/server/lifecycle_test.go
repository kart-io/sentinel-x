package server

import (
	"context"
	"errors"
	"testing"
)

// lifecycleHooks is a test struct implementing lifecycle hooks.
type lifecycleHooks struct {
	startCalled bool
	stopCalled  bool
	startErr    error
	stopErr     error
}

func (h *lifecycleHooks) Start(_ context.Context) error {
	h.startCalled = true
	return h.startErr
}

func (h *lifecycleHooks) Stop(_ context.Context) error {
	h.stopCalled = true
	return h.stopErr
}

func TestLifecycleInterface(_ *testing.T) {
	// Verify that our mock implements the Lifecycle interface
	var _ Lifecycle = (*lifecycleHooks)(nil)
}

func TestLifecycleStart(t *testing.T) {
	tests := []struct {
		name     string
		startErr error
		wantErr  bool
	}{
		{
			name:     "successful start",
			startErr: nil,
			wantErr:  false,
		},
		{
			name:     "start with error",
			startErr: errors.New("start failed"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &lifecycleHooks{
				startErr: tt.startErr,
			}

			ctx := context.Background()
			err := lc.Start(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !lc.startCalled {
				t.Error("Start() was not called")
			}
		})
	}
}

func TestLifecycleStop(t *testing.T) {
	tests := []struct {
		name    string
		stopErr error
		wantErr bool
	}{
		{
			name:    "successful stop",
			stopErr: nil,
			wantErr: false,
		},
		{
			name:    "stop with error",
			stopErr: errors.New("stop failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &lifecycleHooks{
				stopErr: tt.stopErr,
			}

			ctx := context.Background()
			err := lc.Stop(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !lc.stopCalled {
				t.Error("Stop() was not called")
			}
		})
	}
}

func TestLifecycleStartStop(t *testing.T) {
	lc := &lifecycleHooks{}
	ctx := context.Background()

	// Test start
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !lc.startCalled {
		t.Error("Start() was not called")
	}

	// Test stop
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !lc.stopCalled {
		t.Error("Stop() was not called")
	}
}

func TestLifecycleContextCancellation(t *testing.T) {
	lc := &lifecycleHooks{}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Start should still work even with cancelled context
	// (it's up to the implementation to check context)
	err := lc.Start(ctx)
	if err != nil {
		t.Errorf("Start() with cancelled context error = %v", err)
	}

	if !lc.startCalled {
		t.Error("Start() was not called")
	}
}

func TestLifecycleServerAlias(t *testing.T) {
	// Verify that Server is an alias for Lifecycle
	var lc Lifecycle = &lifecycleHooks{}
	srv := lc // 类型会从右侧推导

	ctx := context.Background()

	if err := srv.Start(ctx); err != nil {
		t.Errorf("Server Start() error = %v", err)
	}

	if err := srv.Stop(ctx); err != nil {
		t.Errorf("Server Stop() error = %v", err)
	}
}

func TestRunnableInterface(t *testing.T) {
	// Verify mockRunnable implements Runnable
	runnable := &mockRunnable{name: "test-runnable"}

	var _ Runnable = runnable

	if runnable.Name() != "test-runnable" {
		t.Errorf("Name() = %s, want test-runnable", runnable.Name())
	}
}

func TestRunnableLifecycle(t *testing.T) {
	runnable := &mockRunnable{name: "lifecycle-test"}
	ctx := context.Background()

	// Test Start
	if err := runnable.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !runnable.WasStartCalled() {
		t.Error("Start() was not called")
	}

	// Test Stop
	if err := runnable.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !runnable.WasStopCalled() {
		t.Error("Stop() was not called")
	}
}

func TestRunnableWithErrors(t *testing.T) {
	tests := []struct {
		name     string
		startErr error
		stopErr  error
	}{
		{
			name:     "start error",
			startErr: errors.New("start failed"),
			stopErr:  nil,
		},
		{
			name:     "stop error",
			startErr: nil,
			stopErr:  errors.New("stop failed"),
		},
		{
			name:     "both errors",
			startErr: errors.New("start failed"),
			stopErr:  errors.New("stop failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runnable := &mockRunnable{
				name:     "error-test",
				startErr: tt.startErr,
				stopErr:  tt.stopErr,
			}

			ctx := context.Background()

			// Test Start
			err := runnable.Start(ctx)
			if tt.startErr != nil && err == nil {
				t.Error("Expected start error, got nil")
			}
			if tt.startErr != nil && err.Error() != tt.startErr.Error() {
				t.Errorf("Start() error = %v, want %v", err, tt.startErr)
			}

			// Test Stop
			err = runnable.Stop(ctx)
			if tt.stopErr != nil && err == nil {
				t.Error("Expected stop error, got nil")
			}
			if tt.stopErr != nil && err.Error() != tt.stopErr.Error() {
				t.Errorf("Stop() error = %v, want %v", err, tt.stopErr)
			}
		})
	}
}
