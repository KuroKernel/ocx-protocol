package deterministicvm

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectArtifactFormat tests the artifact format detection
func TestDetectArtifactFormat(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  []byte
		expected string
	}{
		{
			name:     "WASM file",
			filename: "test.wasm",
			content:  []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}, // WASM magic
			expected: "wasm",
		},
		{
			name:     "JavaScript file",
			filename: "test.js",
			content:  []byte("console.log('hello');"),
			expected: "javascript",
		},
		{
			name:     "Python file",
			filename: "test.py",
			content:  []byte("print('hello')"),
			expected: "python",
		},
		{
			name:     "Shell script",
			filename: "test.sh",
			content:  []byte("#!/bin/sh\necho hello"),
			expected: "shell",
		},
		{
			name:     "Bash script",
			filename: "test.bash",
			content:  []byte("#!/bin/bash\necho hello"),
			expected: "bash",
		},
		{
			name:     "ELF binary",
			filename: "test",
			content:  []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}, // ELF magic
			expected: "elf",
		},
		{
			name:     "Windows executable",
			filename: "test.exe",
			content:  []byte("MZ"), // PE magic
			expected: "windows_executable",
		},
		{
			name:     "Shared object",
			filename: "test.so",
			content:  []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}, // ELF magic
			expected: "shared_object",
		},
		{
			name:     "Unknown file",
			filename: "test.unknown",
			content:  []byte("random content"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(filePath, tt.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := detectArtifactFormat(filePath)
			if result != tt.expected {
				t.Errorf("Expected format %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestIsELFBinary tests ELF binary detection
func TestIsELFBinary(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Valid ELF binary",
			content:  []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00},
			expected: true,
		},
		{
			name:     "Invalid ELF binary",
			content:  []byte{0x7f, 'E', 'L', 'F', 0x00, 0x00, 0x00, 0x00},
			expected: true, // Still valid ELF magic
		},
		{
			name:     "Not ELF binary",
			content:  []byte{0x00, 0x61, 0x73, 0x6d}, // WASM magic
			expected: false,
		},
		{
			name:     "Empty file",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test")
			err := os.WriteFile(filePath, tt.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := isELFBinary(filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsWASMBinary tests WASM binary detection
func TestIsWASMBinary(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Valid WASM binary",
			content:  []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Not WASM binary",
			content:  []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}, // ELF magic
			expected: false,
		},
		{
			name:     "Empty file",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test")
			err := os.WriteFile(filePath, tt.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := isWASMBinaryFile(filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsShellScript tests shell script detection
func TestIsShellScript(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Bash script",
			content:  []byte("#!/bin/bash\necho hello"),
			expected: true,
		},
		{
			name:     "Sh script",
			content:  []byte("#!/bin/sh\necho hello"),
			expected: true,
		},
		{
			name:     "Zsh script",
			content:  []byte("#!/bin/zsh\necho hello"),
			expected: true,
		},
		{
			name:     "Fish script",
			content:  []byte("#!/bin/fish\necho hello"),
			expected: true,
		},
		{
			name:     "Env bash script",
			content:  []byte("#!/usr/bin/env bash\necho hello"),
			expected: true,
		},
		{
			name:     "Not a shell script",
			content:  []byte("echo hello"), // No shebang
			expected: false,
		},
		{
			name:     "Empty file",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test")
			err := os.WriteFile(filePath, tt.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := isShellScript(filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
