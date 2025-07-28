package util

// This file provides network utility functions, including ICMP Ping and TCP port checking.
// The code aims for simplicity, readability, and adherence to Go best practices.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// PingHost tests if a host is reachable by sending ICMP echo requests (ping).
// It takes the target IP address and a timeout duration.
// Returns true if a ping reply is received within the timeout, false otherwise.
// Returns an error if the ping command fails to execute or times out.
func PingHost(ip string, timeout time.Duration) (bool, error) {
	if ip == "" {
		return false, fmt.Errorf("IP address cannot be empty")
	}
	if timeout <= 0 {
		timeout = 3 * time.Second // Default timeout if not specified or invalid
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	// Ping arguments differ between OSes.
	// -n/-c: Number of pings to send. 2 is usually sufficient for a quick check.
	// -w/-W: Timeout *per ping* in milliseconds/seconds. Set low (1s) to avoid waiting long for unresponsive hosts.
	// The overall timeout is controlled by the context.
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "2", "-w", "1000", ip)
	} else { // Linux, macOS, other Unix-like
		cmd = exec.CommandContext(ctx, "ping", "-c", "2", "-W", "1", ip)
	}

	output, err := cmd.CombinedOutput()

	// Check for context timeout first
	if ctx.Err() == context.DeadlineExceeded {
		return false, fmt.Errorf("ping %s timed out after %v: %w", ip, timeout, ctx.Err())
	}
	// Check for other command execution errors
	if err != nil {
		// Even if the command exits with an error (e.g., exit code 1),
		// it might still contain output indicating success (e.g., 1 packet received on Linux).
		// So, we don't return immediately here, but we will check the output.
		// However, if there's a fundamental execution error (command not found, etc.),
		// CombinedOutput might return an error *and* empty output.
		// We will implicitly handle this when checking the output below.
		// It's useful to return the underlying error though.
		// Let's refine this: return the error *unless* the output clearly indicates success.
	}

	// Convert potential non-UTF8 output (like GBK on Chinese Windows) to UTF-8
	utf8Output, convErr := convertToUTF8(output)
	if convErr != nil {
		// If conversion fails, fallback to original output but log the conversion error possibility
		// Or return an error immediately? Let's return an error for clarity.
		return false, fmt.Errorf("failed to convert ping output for %s to UTF-8: %w; original error (if any): %v", ip, convErr, err)
	}

	// Check output for success indicators (case-insensitive)
	lowerOutput := strings.ToLower(utf8Output)
	successConditions := []string{
		// Linux/Unix/macOS styles
		"1 received", "2 received", // Short form often seen
		"1 packets received", "2 packets received", // Full form
		"bytes from", // Common indicator of a reply
		// Windows styles (English/Chinese)
		"received = 1", "received = 2",
		"已接收 = 1", "已接收 = 2", // Example for Chinese Windows GBK output
		"来自", // Another common indicator in Chinese replies
	}

	for _, condition := range successConditions {
		if strings.Contains(lowerOutput, condition) {
			return true, nil // Ping successful based on output
		}
	}

	// If we reach here, the command might have finished (err could be nil or non-nil),
	// but the output doesn't indicate success.
	if err != nil {
		// Return the original command execution error
		return false, fmt.Errorf("ping %s command failed: %w; output: %s", ip, err, utf8Output)
	}

	// Command finished without error, but output doesn't look like success
	return false, fmt.Errorf("ping %s command succeeded but output indicates failure: %s", ip, utf8Output)
}

// convertToUTF8 attempts to decode byte slice assumed to be GBK (common on Chinese Windows) into UTF-8 string.
// Falls back to interpreting as UTF-8 if decoding fails.
func convertToUTF8(s []byte) (string, error) {
	// Try decoding as GBK
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader) // Use io.ReadAll (Go 1.16+)
	if err == nil {
		return string(d), nil // Successfully decoded as GBK
	}
	// If GBK decoding failed, return the original bytes interpreted as UTF-8,
	// along with the decoding error for context. It might be UTF-8 already or some other encoding.
	// Returning the original string(s) is often a reasonable fallback.
	return string(s), fmt.Errorf("GBK decoding failed: %w", err)
}

// CheckTCPPort tests if a TCP connection can be established to a specific IP and port within a given timeout.
// Returns true if the connection succeeds, false otherwise.
// Returns an error if the dialing process fails.
func CheckTCPPort(ip string, port int, timeout time.Duration) (bool, error) {
	if ip == "" {
		return false, fmt.Errorf("IP address cannot be empty")
	}
	if port <= 0 || port > 65535 {
		return false, fmt.Errorf("invalid port number: %d", port)
	}
	if timeout <= 0 {
		timeout = 2 * time.Second // Default timeout
	}

	address := net.JoinHostPort(ip, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Wrap the error for more context
		return false, fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	// Don't forget to close the connection if successfully opened!
	defer conn.Close()
	return true, nil // Connection successful
} 