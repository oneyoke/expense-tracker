package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_success.db")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "testuser", "-password", "secret", "-db", dbPath}
	err := run(args, stdin, stdout, stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "User testuser created successfully") {
		t.Errorf("unexpected stdout: %s", output)
	}
}

func TestRun_DuplicateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_duplicate.db")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "testuser", "-password", "secret", "-db", dbPath}

	// First run
	if err := run(args, stdin, stdout, stderr); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Second run
	stdout.Reset()
	stderr.Reset()
	err := run(args, stdin, stdout, stderr)
	if err == nil {
		t.Fatal("expected error on duplicate user, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestRun_MissingUserFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	// Missing user
	args := []string{"-password", "secret"}
	err := run(args, stdin, stdout, stderr)
	if err == nil {
		t.Fatal("expected error for missing user flag, got nil")
	}
	if !strings.Contains(err.Error(), "missing required flags: user") {
		t.Errorf("unexpected error: %v", err)
	}

	// Usage should be printed
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage info in stdout")
	}
}

func TestRun_InteractivePassword(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_interactive.db")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// Simulate user typing "interactive_secret" followed by newline
	stdin := bytes.NewBufferString("interactive_secret\n")

	// Omit -password flag
	args := []string{"-user", "interactive_user", "-db", dbPath}
	err := run(args, stdin, stdout, stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	// Should verify that it prompted for password
	if !strings.Contains(output, "Password: ") {
		t.Errorf("expected password prompt in stdout")
	}
	if !strings.Contains(output, "User interactive_user created successfully") {
		t.Errorf("unexpected stdout: %s", output)
	}
}

func TestRun_InteractivePassword_Empty(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// Simulate user typing newline (empty password)
	stdin := bytes.NewBufferString("\n")

	// Omit -password flag
	args := []string{"-user", "empty_pass_user"}
	err := run(args, stdin, stdout, stderr)
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
	if !strings.Contains(err.Error(), "password cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_EnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env.db")

	os.Setenv("DB_PATH", dbPath)
	defer os.Unsetenv("DB_PATH")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	// Do not pass -db flag, let it use env var
	args := []string{"-user", "envuser", "-password", "secret"}
	err := run(args, stdin, stdout, stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify DB file was created at dbPath
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected DB file at %s, but it does not exist", dbPath)
	}
}

func TestRun_InvalidDBPath(t *testing.T) {
	// Use a directory path as DB file path, which should fail
	tmpDir := t.TempDir()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-user", "failuser", "-password", "secret", "-db", tmpDir}
	err := run(args, stdin, stdout, stderr)
	if err == nil {
		t.Fatal("expected error for invalid db path, got nil")
	}
	// Error message depends on OS/sqlite driver, but should be non-nil
	if !strings.Contains(err.Error(), "failed to open database") {
		t.Errorf("expected 'failed to open database' error, got: %v", err)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdin := new(bytes.Buffer)

	args := []string{"-invalid"}
	err := run(args, stdin, stdout, stderr)
	if err == nil {
		t.Fatal("expected error for invalid flag, got nil")
	}
	// flag package returns error for undefined flags
	// error message usually "flag provided but not defined: -invalid"
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("unexpected error: %v", err)
	}
}
