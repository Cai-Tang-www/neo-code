package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadWriteEditTools(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	writeTool := NewWriteTool()
	readTool := NewReadTool()
	editTool := NewEditTool()

	if _, err := writeTool.Run(ctx, Input{Root: root, Path: "demo.txt", Content: "hello world"}); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	readResult, err := readTool.Run(ctx, Input{Root: root, Path: "demo.txt"})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if readResult.Output != "hello world" {
		t.Fatalf("unexpected read output: %q", readResult.Output)
	}

	editResult, err := editTool.Run(ctx, Input{Root: root, Path: "demo.txt", Old: "world", New: "agent"})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if !strings.Contains(editResult.Output, "hello agent") {
		t.Fatalf("unexpected edit output: %q", editResult.Output)
	}
}

func TestBashTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "demo.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	result, err := NewBashTool(5*time.Second).Run(context.Background(), Input{Root: root, Command: "cat demo.txt"})
	if err != nil {
		t.Fatalf("bash failed: %v", err)
	}
	if !strings.Contains(result.Output, "ok") {
		t.Fatalf("unexpected command output: %q", result.Output)
	}
}

func TestResolvePathRejectsEscape(t *testing.T) {
	root := t.TempDir()
	_, err := NewReadTool().Run(context.Background(), Input{Root: root, Path: "../outside.txt"})
	if err == nil {
		t.Fatal("expected escape path error")
	}
}
