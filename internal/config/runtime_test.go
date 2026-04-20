package config

import "testing"

func TestRuntimeConfigClone(t *testing.T) {
	t.Parallel()

	cfg := RuntimeConfig{
		MaxNoProgressStreak:         7,
		MaxRepeatCycleStreak:        4,
		MaxTurn:                     11,
		SubAgentDispatchConcurrency: 3,
	}
	cloned := cfg.Clone()
	if cloned.MaxNoProgressStreak != 7 {
		t.Fatalf("expected cloned MaxNoProgressStreak=7, got %d", cloned.MaxNoProgressStreak)
	}
	if cloned.MaxRepeatCycleStreak != 4 {
		t.Fatalf("expected cloned MaxRepeatCycleStreak=4, got %d", cloned.MaxRepeatCycleStreak)
	}
	if cloned.MaxTurn != 11 {
		t.Fatalf("expected cloned MaxTurn=11, got %d", cloned.MaxTurn)
	}
	if cloned.SubAgentDispatchConcurrency != 3 {
		t.Fatalf(
			"expected cloned SubAgentDispatchConcurrency=3, got %d",
			cloned.SubAgentDispatchConcurrency,
		)
	}
	if cloned.Assets.MaxSessionAssetBytes != cfg.Assets.MaxSessionAssetBytes {
		t.Fatalf("expected cloned MaxSessionAssetBytes=%d, got %d", cfg.Assets.MaxSessionAssetBytes, cloned.Assets.MaxSessionAssetBytes)
	}
	if cloned.Assets.MaxSessionAssetsTotalBytes != cfg.Assets.MaxSessionAssetsTotalBytes {
		t.Fatalf(
			"expected cloned MaxSessionAssetsTotalBytes=%d, got %d",
			cfg.Assets.MaxSessionAssetsTotalBytes,
			cloned.Assets.MaxSessionAssetsTotalBytes,
		)
	}
}

func TestRuntimeConfigApplyDefaults(t *testing.T) {
	t.Parallel()

	defaults := RuntimeConfig{
		MaxNoProgressStreak:         5,
		MaxRepeatCycleStreak:        5,
		MaxTurn:                     20,
		SubAgentDispatchConcurrency: 2,
		Assets: RuntimeAssetsConfig{
			MaxSessionAssetBytes:       11,
			MaxSessionAssetsTotalBytes: 22,
		},
	}

	cfg := RuntimeConfig{MaxNoProgressStreak: 0, MaxRepeatCycleStreak: 0, MaxTurn: 0, SubAgentDispatchConcurrency: 0}
	cfg.ApplyDefaults(defaults)
	if cfg.MaxNoProgressStreak != 5 {
		t.Fatalf("expected defaulted MaxNoProgressStreak=5, got %d", cfg.MaxNoProgressStreak)
	}
	if cfg.MaxRepeatCycleStreak != 5 {
		t.Fatalf("expected defaulted MaxRepeatCycleStreak=5, got %d", cfg.MaxRepeatCycleStreak)
	}
	if cfg.MaxTurn != 20 {
		t.Fatalf("expected defaulted MaxTurn=20, got %d", cfg.MaxTurn)
	}
	if cfg.SubAgentDispatchConcurrency != 2 {
		t.Fatalf(
			"expected defaulted SubAgentDispatchConcurrency=2, got %d",
			cfg.SubAgentDispatchConcurrency,
		)
	}
	if cfg.Assets.MaxSessionAssetBytes != 11 {
		t.Fatalf("expected defaulted MaxSessionAssetBytes=11, got %d", cfg.Assets.MaxSessionAssetBytes)
	}
	if cfg.Assets.MaxSessionAssetsTotalBytes != 22 {
		t.Fatalf("expected defaulted MaxSessionAssetsTotalBytes=22, got %d", cfg.Assets.MaxSessionAssetsTotalBytes)
	}

	cfg = RuntimeConfig{
		MaxNoProgressStreak:         5,
		MaxRepeatCycleStreak:        8,
		MaxTurn:                     22,
		SubAgentDispatchConcurrency: 4,
	}
	cfg.ApplyDefaults(defaults)
	if cfg.MaxNoProgressStreak != 5 {
		t.Fatalf("expected existing MaxNoProgressStreak=5 to be preserved, got %d", cfg.MaxNoProgressStreak)
	}
	if cfg.MaxRepeatCycleStreak != 8 {
		t.Fatalf("expected existing MaxRepeatCycleStreak=8 to be preserved, got %d", cfg.MaxRepeatCycleStreak)
	}
	if cfg.MaxTurn != 22 {
		t.Fatalf("expected existing MaxTurn=22 to be preserved, got %d", cfg.MaxTurn)
	}
	if cfg.SubAgentDispatchConcurrency != 4 {
		t.Fatalf(
			"expected existing SubAgentDispatchConcurrency=4 to be preserved, got %d",
			cfg.SubAgentDispatchConcurrency,
		)
	}

	cfg = RuntimeConfig{
		MaxNoProgressStreak:         2,
		MaxRepeatCycleStreak:        -1,
		MaxTurn:                     -1,
		SubAgentDispatchConcurrency: -1,
	}
	cfg.ApplyDefaults(defaults)
	if cfg.MaxRepeatCycleStreak != 5 {
		t.Fatalf("expected negative MaxRepeatCycleStreak=-1 to be replaced by default=5, got %d", cfg.MaxRepeatCycleStreak)
	}
	if cfg.MaxTurn != 20 {
		t.Fatalf("expected negative MaxTurn=-1 to be replaced by default=20, got %d", cfg.MaxTurn)
	}
	if cfg.SubAgentDispatchConcurrency != 2 {
		t.Fatalf(
			"expected negative SubAgentDispatchConcurrency=-1 to be replaced by default=2, got %d",
			cfg.SubAgentDispatchConcurrency,
		)
	}

	var nilCfg *RuntimeConfig
	nilCfg.ApplyDefaults(defaults)
}

func TestRuntimeConfigValidate(t *testing.T) {
	t.Parallel()

	if err := (RuntimeConfig{
		MaxNoProgressStreak:         1,
		MaxRepeatCycleStreak:        1,
		MaxTurn:                     1,
		SubAgentDispatchConcurrency: 1,
	}).Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	for _, bad := range []int{0, -1, -99} {
		if err := (RuntimeConfig{
			MaxNoProgressStreak:         bad,
			MaxRepeatCycleStreak:        1,
			MaxTurn:                     1,
			SubAgentDispatchConcurrency: 1,
		}).Validate(); err == nil {
			t.Fatalf("expected validation error for MaxNoProgressStreak=%d", bad)
		}
	}

	for _, bad := range []int{0, -1, -99} {
		if err := (RuntimeConfig{
			MaxNoProgressStreak:         1,
			MaxRepeatCycleStreak:        bad,
			MaxTurn:                     1,
			SubAgentDispatchConcurrency: 1,
		}).Validate(); err == nil {
			t.Fatalf("expected validation error for MaxRepeatCycleStreak=%d", bad)
		}
	}

	for _, bad := range []int{0, -1, -99} {
		if err := (RuntimeConfig{
			MaxNoProgressStreak:         1,
			MaxRepeatCycleStreak:        1,
			MaxTurn:                     bad,
			SubAgentDispatchConcurrency: 1,
		}).Validate(); err == nil {
			t.Fatalf("expected validation error for MaxTurn=%d", bad)
		}
	}

	for _, bad := range []int{0, -1, -99} {
		if err := (RuntimeConfig{
			MaxNoProgressStreak:         1,
			MaxRepeatCycleStreak:        1,
			MaxTurn:                     1,
			SubAgentDispatchConcurrency: bad,
		}).Validate(); err == nil {
			t.Fatalf("expected validation error for SubAgentDispatchConcurrency=%d", bad)
		}
	}

	if err := (RuntimeConfig{
		MaxNoProgressStreak:         1,
		MaxRepeatCycleStreak:        1,
		MaxTurn:                     1,
		SubAgentDispatchConcurrency: 1,
		Assets: RuntimeAssetsConfig{
			MaxSessionAssetBytes:       1,
			MaxSessionAssetsTotalBytes: 1,
		},
	}).Validate(); err != nil {
		t.Fatalf("expected valid assets config, got %v", err)
	}
	if err := (RuntimeConfig{
		MaxNoProgressStreak:         1,
		MaxRepeatCycleStreak:        1,
		MaxTurn:                     1,
		SubAgentDispatchConcurrency: 1,
		Assets: RuntimeAssetsConfig{
			MaxSessionAssetBytes:       -1,
			MaxSessionAssetsTotalBytes: 1,
		},
	}).Validate(); err == nil {
		t.Fatal("expected validation error for assets.max_session_asset_bytes=-1")
	}
	if err := (RuntimeConfig{
		MaxNoProgressStreak:         1,
		MaxRepeatCycleStreak:        1,
		MaxTurn:                     1,
		SubAgentDispatchConcurrency: 1,
		Assets: RuntimeAssetsConfig{
			MaxSessionAssetBytes:       1,
			MaxSessionAssetsTotalBytes: -1,
		},
	}).Validate(); err == nil {
		t.Fatal("expected validation error for assets.max_session_assets_total_bytes=-1")
	}
}

func TestRuntimeAssetsConfigZeroValuesResolveToDefaults(t *testing.T) {
	t.Parallel()

	cfg := RuntimeAssetsConfig{
		MaxSessionAssetBytes:       0,
		MaxSessionAssetsTotalBytes: 0,
	}
	resolved := cfg.ResolveSessionAssetLimits()
	defaults := defaultRuntimeAssetsConfig()
	if resolved.MaxSessionAssetBytes != defaults.MaxSessionAssetBytes {
		t.Fatalf(
			"expected MaxSessionAssetBytes to fallback to default=%d, got %d",
			defaults.MaxSessionAssetBytes,
			resolved.MaxSessionAssetBytes,
		)
	}
	if resolved.MaxSessionAssetsTotalBytes != defaults.MaxSessionAssetsTotalBytes {
		t.Fatalf(
			"expected MaxSessionAssetsTotalBytes to fallback to default=%d, got %d",
			defaults.MaxSessionAssetsTotalBytes,
			resolved.MaxSessionAssetsTotalBytes,
		)
	}
}
