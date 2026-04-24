package runtime

import (
	"context"
	"testing"

	"neo-code/internal/config"
	providertypes "neo-code/internal/provider/types"
	agentsession "neo-code/internal/session"
)

func TestBeforeAcceptFinalDecisionPaths(t *testing.T) {
	t.Parallel()

	service := &Service{}
	baseCfg := config.StaticDefaults().Clone()
	baseCfg.Runtime.Verification.Enabled = boolPtr(true)
	baseCfg.Runtime.Verification.FinalIntercept = boolPtr(true)
	snapshot := TurnBudgetSnapshot{
		Config:  baseCfg,
		Workdir: t.TempDir(),
	}

	t.Run("pending required todo -> continue", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-continue", agentsession.New("continue"))
		required := true
		state.session.Todos = []agentsession.TodoItem{
			{
				ID:       "todo-1",
				Content:  "do work",
				Status:   agentsession.TodoStatusPending,
				Required: &required,
			},
		}
		decision, err := service.beforeAcceptFinal(context.Background(), &state, snapshot, providertypes.Message{
			Role:  providertypes.RoleAssistant,
			Parts: []providertypes.ContentPart{providertypes.NewTextPart("done")},
		}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.Status != "continue" {
			t.Fatalf("status = %q, want continue", decision.Status)
		}
	})

	t.Run("all converged -> accepted", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-accepted", agentsession.New("accepted"))
		decision, err := service.beforeAcceptFinal(context.Background(), &state, snapshot, providertypes.Message{
			Role:  providertypes.RoleAssistant,
			Parts: []providertypes.ContentPart{providertypes.NewTextPart("done")},
		}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.Status != "accepted" {
			t.Fatalf("status = %q, want accepted", decision.Status)
		}
	})

	t.Run("verification disabled -> compatibility fallback", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-fallback", agentsession.New("fallback"))
		cfg := snapshot.Config
		cfg.Runtime.Verification.Enabled = boolPtr(false)
		assertCompatibilityFallback(t, service, &state, cfg, snapshot.Workdir)
	})

	t.Run("final intercept disabled -> compatibility fallback", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-no-intercept", agentsession.New("no-intercept"))
		cfg := snapshot.Config
		cfg.Runtime.Verification.FinalIntercept = boolPtr(false)
		assertCompatibilityFallback(t, service, &state, cfg, snapshot.Workdir)
	})
}

func TestInferTaskTypeSupportsChineseKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		taskID   string
		goal     string
		nextStep string
		want     string
	}{
		{
			name:     "fix bug chinese",
			taskID:   "task-1",
			goal:     "修复登录失败问题",
			nextStep: "定位故障根因",
			want:     "fix_bug",
		},
		{
			name:     "refactor chinese",
			taskID:   "task-2",
			goal:     "重构 runtime 验收流程",
			nextStep: "拆分函数",
			want:     "refactor",
		},
		{
			name:     "docs chinese",
			taskID:   "task-3",
			goal:     "补充文档",
			nextStep: "更新说明",
			want:     "docs",
		},
		{
			name:     "config chinese",
			taskID:   "task-4",
			goal:     "调整配置",
			nextStep: "更新 yaml",
			want:     "config",
		},
		{
			name:     "create file chinese",
			taskID:   "task-5",
			goal:     "新增文件",
			nextStep: "创建文件并写入模板",
			want:     "create_file",
		},
		{
			name:     "edit code chinese",
			taskID:   "task-6",
			goal:     "修改代码",
			nextStep: "打补丁",
			want:     "edit_code",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			session := agentsession.New(tc.taskID)
			session.TaskState.Goal = tc.goal
			session.TaskState.NextStep = tc.nextStep
			state := newRunState("run-"+tc.taskID, session)
			state.taskID = tc.taskID

			got := inferTaskType(&state)
			if got != tc.want {
				t.Fatalf("inferTaskType() = %q, want %q", got, tc.want)
			}
		})
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func assertCompatibilityFallback(
	t *testing.T,
	service *Service,
	state *runState,
	cfg config.Config,
	workdir string,
) {
	t.Helper()

	decision, err := service.beforeAcceptFinal(context.Background(), state, TurnBudgetSnapshot{
		Config:  cfg,
		Workdir: workdir,
	}, providertypes.Message{}, true)
	if err != nil {
		t.Fatalf("beforeAcceptFinal() error = %v", err)
	}
	if decision.StopReason != "compatibility_fallback" {
		t.Fatalf("stop_reason = %q, want compatibility_fallback", decision.StopReason)
	}
}
