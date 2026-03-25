package service

import (
	"context"
	"strings"
	"testing"

	"go-llm-demo/internal/server/infra/repository"
)

func TestMemoryServiceSavesAndRecallsPersistentMemory(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/memory.json"

	svc := NewMemoryService(
		repository.NewFileMemoryStore(path, 100),
		repository.NewSessionMemoryStore(100),
		5,
		2.2,
		1800,
		path,
		[]string{"user_preference", "project_rule", "code_fact", "fix_recipe"},
	)

	if err := svc.Save(ctx, "以后回答中文，命令和说明都用中文。", "好的，后续我会统一使用中文回复。"); err != nil {
		t.Fatalf("save preference: %v", err)
	}
	if err := svc.Save(ctx, "memory_repository.go 是做什么的？", "internal/server/infra/repository/memory_repository.go 负责长期记忆的文件存储与读写。"); err != nil {
		t.Fatalf("save code fact: %v", err)
	}

	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.PersistentItems != 2 {
		t.Fatalf("expected 2 persistent items, got %+v", stats)
	}
	if stats.ByType["user_preference"] != 1 {
		t.Fatalf("expected 1 user_preference, got %+v", stats.ByType)
	}
	if stats.ByType["code_fact"] != 1 {
		t.Fatalf("expected 1 code_fact, got %+v", stats.ByType)
	}
	if stats.ByType["project_rule"] != 0 {
		t.Fatalf("expected no project_rule for preference-only input, got %+v", stats.ByType)
	}

	reloaded := NewMemoryService(
		repository.NewFileMemoryStore(path, 100),
		repository.NewSessionMemoryStore(100),
		5,
		2.2,
		1800,
		path,
		[]string{"user_preference", "project_rule", "code_fact", "fix_recipe"},
	)

	prompt, err := reloaded.BuildContext(ctx, "请继续用中文，并看看 memory_repository.go 的职责")
	if err != nil {
		t.Fatalf("build context: %v", err)
	}
	if !strings.Contains(prompt, "user_preference") {
		t.Fatalf("expected recalled user_preference in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "code_fact") {
		t.Fatalf("expected recalled code_fact in prompt, got %q", prompt)
	}
}

func TestMemoryServiceSkipsToolCallPayload(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/memory.json"

	svc := NewMemoryService(
		repository.NewFileMemoryStore(path, 100),
		repository.NewSessionMemoryStore(100),
		5,
		2.2,
		1800,
		path,
		[]string{"user_preference", "project_rule", "code_fact", "fix_recipe"},
	)

	reply := `{"tool":"read","params":{"filePath":"internal/server/service/memory_service.go"}}`
	if err := svc.Save(ctx, "请读取 memory_service.go 看看记忆模块怎么实现的。", reply); err != nil {
		t.Fatalf("save tool payload: %v", err)
	}

	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.TotalItems != 0 {
		t.Fatalf("expected tool payload to be skipped, got %+v", stats)
	}
}
