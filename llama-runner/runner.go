package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Runner struct {
	cmd          *exec.Cmd
	internalPort int
}

func StartRunner(binary string, args []string, internalPort int) (*Runner, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("spawning: %s %v", binary, args)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn llama-server: %w", err)
	}

	r := &Runner{cmd: cmd, internalPort: internalPort}
	if err := r.waitReady(2 * time.Minute); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	log.Printf("llama-server ready on internal port %d", internalPort)
	return r, nil
}

func (r *Runner) waitReady(timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", r.internalPort)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("llama-server did not become ready within %v", timeout)
}

func (r *Runner) Kill() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
	}
}
