package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const ExecutionTimeout = 30 * time.Second

// Wrapper template for user code
// Uses fmt.Sprintf to avoid brace escaping issues
func getWrapperCode(userCode string) string {
	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"
)

%s

func main() {
	var event map[string]interface{}
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %%v\n", err)
		os.Exit(1)
	}

	result := handler(event)

	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding output: %%v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
}
`, userCode)
}

// RunCode executes Go code in a sandbox
func RunCode(code string, inputData map[string]interface{}) (status, output, logs string) {
	status = "SUCCESS"

	// Create temporary work directory
	workDir := filepath.Join("/tmp/sandbox", uuid.New().String())
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "ERROR", fmt.Sprintf("Failed to create work directory: %v", err), ""
	}
	defer os.RemoveAll(workDir)

	// Create main.go with wrapped user code
	fullCode := getWrapperCode(code)
	sourceFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(sourceFile, []byte(fullCode), 0644); err != nil {
		return "ERROR", fmt.Sprintf("Failed to write source file: %v", err), ""
	}

	// Prepare input JSON
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return "ERROR", fmt.Sprintf("Failed to marshal input: %v", err), ""
	}

	// Run with go run
	cmd := exec.Command("go", "run", sourceFile)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"GOCACHE=/tmp/gocache",
		"GOPATH=/tmp/gopath",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(inputJSON)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create a channel for completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait with timeout
	select {
	case err := <-done:
		logs = stderr.String()
		if err != nil {
			status = "ERROR"
			output = stderr.String()
			if output == "" {
				output = fmt.Sprintf("Execution failed: %v", err)
			}
		} else {
			output = string(bytes.TrimSpace(stdout.Bytes()))
			status = "SUCCESS"
		}
	case <-time.After(ExecutionTimeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		status = "TIMEOUT"
		output = fmt.Sprintf("Execution timed out after %v", ExecutionTimeout)
	}

	return status, output, logs
}
