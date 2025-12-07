package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getLocalNameByPort(port int) (string, error) {
	if port <= 0 {
		return "", fmt.Errorf("empty or invalid port")
	}

	pid, err := findPIDWithSS(port)
	if err != nil {
		return "", fmt.Errorf("find process by inode: %w", err)
	}

	return getExecutableName(pid)
}

func execSS() []string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, "ss", "-tanp")
	var out bytes.Buffer
	c.Stdout = &out

	var errout strings.Builder
	c.Stderr = &errout

	err := c.Run()
	if err != nil {
		fmt.Printf("execute ss: %v\n", errout.String())
		return nil
	}

	var lines []string
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}
	err = sc.Err()
	if err != nil {
		fmt.Printf("scan ss output: %v\n", err)
		return nil
	}

	return lines
}

// findPIDWithSS uses ss command to find process ID
func findPIDWithSS(port int) (int, error) {
	lines := execSS()
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(":%d", port)) {
			// Extract PID from ss output
			// Format: LISTEN 0 128 *:443 *:* users:(("nginx",pid=1234,fd=3))
			pidStart := strings.Index(line, "pid=")
			if pidStart == -1 {
				continue
			}

			pidStr := ""
			for i := pidStart + 4; i < len(line); i++ {
				if line[i] < '0' || line[i] > '9' {
					break
				}
				pidStr += string(line[i])
			}

			if pidStr != "" {
				pid, err := strconv.Atoi(pidStr)
				if err == nil {
					return pid, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("port %d not found in ss output", port)
}

// getExecutableName gets the executable name from a process ID
func getExecutableName(pid int) (string, error) {
	// Try reading the exe symlink
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	target, err := os.Readlink(exePath)
	if err != nil {
		// Fallback to reading the cmdline
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			return "", fmt.Errorf("failed to get executable name for pid %d: %w", pid, err)
		}

		// cmdline is null-separated, first element is the executable
		parts := strings.Split(string(cmdline), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			return filepath.Base(parts[0]), nil
		}

		return "", fmt.Errorf("empty cmdline for pid %d", pid)
	}

	return filepath.Base(target), nil
}
