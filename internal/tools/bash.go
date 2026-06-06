package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yanking/go-tiny-claw/internal/schema"
)

var (
	bgMu      sync.Mutex
	bgProcSeq int64
	bgProcs   = make(map[int64]*bgProcess)
)

type bgProcess struct {
	pid     int
	logFile string
	cmd     string
	startAt time.Time
}

type BashTool struct {
	workDir string
	timeout time.Duration
}

func NewBashTool(workDir string) *BashTool {
	return &BashTool{
		workDir: workDir,
		timeout: 30 * time.Second,
	}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: fmt.Sprintf("Execute a shell command in the workspace. OS: %s, Shell: %s.", runtime.GOOS, t.shellName()),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"run_in_background": map[string]interface{}{
					"type":        "boolean",
					"description": "Run in background (nohup). Output goes to a log file, returns immediately with PID and log path.",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) shellName() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

type bashArgs struct {
	Command         string `json:"command"`
	RunInBackground bool   `json:"run_in_background"`
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input bashArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if input.RunInBackground {
		return t.runBackground(input.Command)
	}
	return t.runForeground(ctx, input.Command)
}

func (t *BashTool) runForeground(ctx context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := t.buildCmd(ctx, command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := mergeOutput(stdout.String(), stderr.String())
	output = truncateOutput(output)

	if err != nil {
		if output == "" {
			output = err.Error()
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	if output == "" {
		return "(command completed, no output)", nil
	}
	return output, nil
}

func (t *BashTool) runBackground(command string) (string, error) {
	seq := atomic.AddInt64(&bgProcSeq, 1)

	logDir := filepath.Join(t.workDir, ".claw", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("nohup-%d.log", seq))
	logFH, err := os.Create(logFile)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}

	cmd := t.buildCmd(context.Background(), command)
	cmd.Stdout = logFH
	cmd.Stderr = logFH
	cmd.SysProcAttr = newProcessGroupAttr()

	if err := cmd.Start(); err != nil {
		logFH.Close()
		return "", fmt.Errorf("start background process: %w", err)
	}

	pid := cmd.Process.Pid
	bgMu.Lock()
	bgProcs[seq] = &bgProcess{
		pid:     pid,
		logFile: logFile,
		cmd:     command,
		startAt: time.Now(),
	}
	bgMu.Unlock()

	go func() {
		cmd.Wait()
		logFH.Close()
	}()

	return fmt.Sprintf("Background process started.\n  Seq: %d  PID: %d\n  Log: %s\n  Cmd: %s\n\nUse read_file to check the log.",
		seq, pid, logFile, command), nil
}

func (t *BashTool) buildCmd(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func mergeOutput(stdout, stderr string) string {
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

func truncateOutput(output string) string {
	const maxLen = 8000
	if len(output) > maxLen {
		return output[:maxLen] + "\n...(truncated)"
	}
	return output
}
