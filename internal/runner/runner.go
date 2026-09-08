package runner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

const (
	outputBufferSize = 100
	scannerCount     = 2
)

// Runner manages the execution of a command and its I/O.
type Runner struct {
	cmd        *exec.Cmd
	stdin      chan string
	outputChan chan string   // Channel for streaming output
	done       chan struct{} // Channel to signal when the command is done
	stdinOnce  sync.Once
}

// New creates a new Runner instance.
func New(command string, args ...string) *Runner {
	return &Runner{
		cmd:        exec.CommandContext(context.Background(), command, args...),
		stdin:      make(chan string),
		outputChan: make(chan string, outputBufferSize), // Buffered channel for output.
		done:       make(chan struct{}),
	}
}

// Start begins the command execution and sets up I/O handling.
func (r *Runner) Start() error {
	// Create stdin pipe.
	stdinPipe, err := r.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating stdin pipe: %w", err)
	}

	// Create stdout pipe.
	stdoutPipe, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	// Create stderr pipe.
	stderrPipe, err := r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	// Start command.
	if startErr := r.cmd.Start(); startErr != nil {
		return fmt.Errorf("error starting command: %w", startErr)
	}

	// Create scanners for stdout and stderr
	outScanner := bufio.NewScanner(stdoutPipe)
	errScanner := bufio.NewScanner(stderrPipe)

	// Create a WaitGroup to coordinate the scanner goroutines.
	var scanners sync.WaitGroup
	scanners.Add(scannerCount)

	// Start goroutine to scan stdout
	go func() {
		defer scanners.Done()
		for outScanner.Scan() {
			select {
			case r.outputChan <- outScanner.Text():
			default:
				// Channel is full, discard output
			}
		}
	}()

	// Start goroutine to scan stderr
	go func() {
		defer scanners.Done()
		for errScanner.Scan() {
			select {
			case r.outputChan <- "[ERR] " + errScanner.Text():
			default:
				// Channel is full, discard output
			}
		}
	}()

	// Start goroutine to manage output channel closure
	go func() {
		scanners.Wait()     // Wait for both scanners to complete
		close(r.outputChan) // Then close the output channel
	}()

	// Start goroutine to forward input to the process
	go func() {
		defer stdinPipe.Close() // Ensure stdin is closed when done.
		for input := range r.stdin {
			input += "\n"
			if _, writeErr := stdinPipe.Write([]byte(input)); writeErr != nil {
				fmt.Fprintf(os.Stderr, "Error writing to stdin: %v\n", writeErr)
				return
			}
		}
	}()

	return nil
}

// Close shuts down stdin for the running command.
func (r *Runner) Close() {
	r.stdinOnce.Do(func() {
		close(r.stdin)
	})
}

// WriteInput sends input to the running command.
func (r *Runner) WriteInput(input string) {
	r.stdin <- input
}

// GetOutputChan returns a channel that receives command output in real-time.
func (r *Runner) GetOutputChan() <-chan string {
	return r.outputChan
}

// Done returns a channel that's closed when the command completes.
func (r *Runner) Done() <-chan struct{} {
	return r.done
}

// Wait waits for the command to complete.
func (r *Runner) Wait() error {
	return r.cmd.Wait()
}
