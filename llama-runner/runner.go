package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Runner struct {
	binary       string
	args         []string
	internalPort int

	mu     sync.Mutex
	cmd    *exec.Cmd
	ctx    context.Context
	cancel context.CancelFunc
}

func StartRunner(binary string, args []string, internalPort int) (*Runner, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Runner{
		binary:       binary,
		args:         args,
		internalPort: internalPort,
		ctx:          ctx,
		cancel:       cancel,
	}
	if err := r.start(); err != nil {
		cancel()
		return nil, err
	}
	return r, nil
}

func (r *Runner) start() error {
	cmd := exec.Command(r.binary, r.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("spawning: %s %v", r.binary, r.args)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn llama-server: %w", err)
	}
	if err := r.waitReady(2 * time.Minute); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	r.cmd = cmd
	log.Printf("llama-server ready on internal port %d", r.internalPort)
	return nil
}

// EnsureHealthy checks if llama-server is responding and restarts it if not.
func (r *Runner) EnsureHealthy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isHealthy() {
		return nil
	}

	log.Printf("llama-server unhealthy; restarting")
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
		r.cmd = nil
	}
	return r.start()
}

func (r *Runner) isHealthy() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", r.internalPort)
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (r *Runner) waitReady(timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", r.internalPort)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		default:
		}
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("llama-server did not become ready within %v", timeout)
}

func (r *Runner) Kill() {
	r.cancel()
	r.mu.Lock()
	cmd := r.cmd
	r.cmd = nil
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}
