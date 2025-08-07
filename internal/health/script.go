package health

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
)

// ScriptHealthChecker implements a health checker that runs custom scripts
type ScriptHealthChecker struct {
	name         string
	scriptPath   string
	args         []string
	timeout      time.Duration
	shell        string
	successCodes []int
}

// NewScriptHealthChecker creates a new script health checker
func NewScriptHealthChecker(name, scriptPath string, timeout time.Duration) *ScriptHealthChecker {
	return &ScriptHealthChecker{
		name:       name,
		scriptPath: scriptPath,
		timeout:    timeout,
		shell:      "",
		args:       []string{},
		// By default, only exit code 0 is considered successful
		successCodes: []int{0},
	}
}

// Name returns the name of the health checker
func (s *ScriptHealthChecker) Name() string {
	return s.name
}

// Type returns the type of the health checker
func (s *ScriptHealthChecker) Type() string {
	return "script"
}

// Configure configures the health checker with the given options
func (s *ScriptHealthChecker) Configure(options map[string]interface{}) error {
	if scriptPath, ok := options["scriptPath"].(string); ok {
		s.scriptPath = scriptPath
	}

	if timeout, ok := options["timeout"].(time.Duration); ok {
		s.timeout = timeout
	}

	if shell, ok := options["shell"].(string); ok {
		s.shell = shell
	}

	if args, ok := options["args"].([]string); ok {
		s.args = args
	} else if argsInterface, ok := options["args"].([]interface{}); ok {
		// Convert []interface{} to []string
		args := make([]string, len(argsInterface))
		for i, arg := range argsInterface {
			if strArg, ok := arg.(string); ok {
				args[i] = strArg
			} else {
				args[i] = fmt.Sprintf("%v", arg)
			}
		}
		s.args = args
	}

	if successCodes, ok := options["successCodes"].([]int); ok {
		s.successCodes = successCodes
	} else if successCodesInterface, ok := options["successCodes"].([]interface{}); ok {
		// Convert []interface{} to []int
		successCodes := make([]int, len(successCodesInterface))
		for i, code := range successCodesInterface {
			if intCode, ok := code.(int); ok {
				successCodes[i] = intCode
			} else if floatCode, ok := code.(float64); ok {
				successCodes[i] = int(floatCode)
			} else {
				return fmt.Errorf("invalid success code: %v", code)
			}
		}
		s.successCodes = successCodes
	}

	return nil
}

// Check performs a health check and returns the result
func (s *ScriptHealthChecker) Check() (*plugin.HealthCheckResult, error) {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Prepare command
	var cmd *exec.Cmd
	if s.shell != "" {
		// If shell is specified, run the script through the shell
		shellArgs := []string{"-c", fmt.Sprintf("%s %s", s.scriptPath, strings.Join(s.args, " "))}
		cmd = exec.CommandContext(ctx, s.shell, shellArgs...)
	} else {
		// Otherwise, run the script directly
		cmd = exec.CommandContext(ctx, s.scriptPath, s.args...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Calculate duration
	duration := time.Since(startTime).Milliseconds()

	// Check if the command timed out
	if ctx.Err() == context.DeadlineExceeded {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("Script execution timed out after %d ms", duration),
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms": fmt.Sprintf("%d", duration),
				"stdout":      stdout.String(),
				"stderr":      stderr.String(),
			},
		}, nil
	}

	// Check if the command failed
	if err != nil {
		// Check if it's an exit error
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			exitCode := exitErr.ExitCode()

			// Check if the exit code is in the list of success codes
			for _, code := range s.successCodes {
				if exitCode == code {
					return &plugin.HealthCheckResult{
						Status:    plugin.StatusUp,
						Message:   fmt.Sprintf("Script executed successfully with exit code %d", exitCode),
						Timestamp: time.Now().Unix(),
						Metadata: map[string]string{
							"duration_ms": fmt.Sprintf("%d", duration),
							"exit_code":   fmt.Sprintf("%d", exitCode),
							"stdout":      stdout.String(),
							"stderr":      stderr.String(),
						},
					}, nil
				}
			}

			return &plugin.HealthCheckResult{
				Status:    plugin.StatusDown,
				Message:   fmt.Sprintf("Script execution failed with exit code %d", exitCode),
				Timestamp: time.Now().Unix(),
				Metadata: map[string]string{
					"duration_ms": fmt.Sprintf("%d", duration),
					"exit_code":   fmt.Sprintf("%d", exitCode),
					"stdout":      stdout.String(),
					"stderr":      stderr.String(),
				},
			}, nil
		}

		// If it's not an exit error, it's some other error
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("Script execution failed: %v", err),
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms": fmt.Sprintf("%d", duration),
				"error":       err.Error(),
				"stdout":      stdout.String(),
				"stderr":      stderr.String(),
			},
		}, nil
	}

	// Command succeeded
	logger.Debug("Script executed successfully: %s", s.scriptPath)
	return &plugin.HealthCheckResult{
		Status:    plugin.StatusUp,
		Message:   "Script executed successfully",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"duration_ms": fmt.Sprintf("%d", duration),
			"exit_code":   "0",
			"stdout":      stdout.String(),
			"stderr":      stderr.String(),
		},
	}, nil
}
