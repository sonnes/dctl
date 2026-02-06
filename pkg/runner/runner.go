package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ContainerBin is the path to the container CLI binary.
var ContainerBin = findContainerBin()

func findContainerBin() string {
	if bin := os.Getenv("DCTL_CONTAINER_BIN"); bin != "" {
		return bin
	}
	if path, err := exec.LookPath("container"); err == nil {
		return path
	}
	for _, p := range []string{"/usr/local/bin/container"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "container"
}

// Run executes a container CLI command, streaming stdin/stdout/stderr.
func Run(args ...string) error {
	cmd := exec.Command(ContainerBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// Output executes a container CLI command and captures stdout.
func Output(args ...string) (string, error) {
	cmd := exec.Command(ContainerBin, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Exec replaces the current process with the container CLI.
func Exec(args ...string) error {
	binary, err := exec.LookPath(ContainerBin)
	if err != nil {
		return fmt.Errorf("container binary not found: %w", err)
	}
	argv := append([]string{"container"}, args...)
	return syscall.Exec(binary, argv, os.Environ())
}

// BuildArgs constructs a container CLI argument list from flag mappings.
// It skips empty values and handles repeated flags (e.g. -e for env).
func BuildArgs(base []string, flags map[string]string, sliceFlags map[string][]string, boolFlags map[string]bool) []string {
	args := make([]string, len(base))
	copy(args, base)

	for flag, val := range flags {
		if val != "" {
			args = append(args, flag, val)
		}
	}
	for flag, vals := range sliceFlags {
		for _, v := range vals {
			args = append(args, flag, v)
		}
	}
	for flag, val := range boolFlags {
		if val {
			args = append(args, flag)
		}
	}
	return args
}
