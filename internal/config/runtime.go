package config

import (
	"errors"
)

const (
	DefaultMaxNoProgressStreak         = 5
	DefaultMaxRepeatCycleStreak        = 3
	DefaultMaxTurn                     = 20
	DefaultSubAgentDispatchConcurrency = 2
)

// RuntimeConfig 定义 runtime 层的可调参数。
type RuntimeConfig struct {
	MaxNoProgressStreak         int `yaml:"max_no_progress_streak,omitempty"`
	MaxRepeatCycleStreak        int `yaml:"max_repeat_cycle_streak,omitempty"`
	MaxTurn                     int `yaml:"max_turn,omitempty"`
	SubAgentDispatchConcurrency int `yaml:"subagent_dispatch_concurrency,omitempty"`
}

// defaultRuntimeConfig 返回 runtime 配置的静态默认值。
func defaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		MaxNoProgressStreak:         DefaultMaxNoProgressStreak,
		MaxRepeatCycleStreak:        DefaultMaxRepeatCycleStreak,
		MaxTurn:                     DefaultMaxTurn,
		SubAgentDispatchConcurrency: DefaultSubAgentDispatchConcurrency,
	}
}

// Clone 复制 runtime 配置，避免调用方共享可变状态。
func (c RuntimeConfig) Clone() RuntimeConfig {
	return c
}

// ApplyDefaults 在配置缺失或非法时回填默认阈值。
func (c *RuntimeConfig) ApplyDefaults(defaults RuntimeConfig) {
	if c == nil {
		return
	}
	if c.MaxNoProgressStreak <= 0 {
		c.MaxNoProgressStreak = defaults.MaxNoProgressStreak
	}
	if c.MaxRepeatCycleStreak <= 0 {
		c.MaxRepeatCycleStreak = defaults.MaxRepeatCycleStreak
	}
	if c.MaxTurn <= 0 {
		c.MaxTurn = defaults.MaxTurn
	}
	if c.SubAgentDispatchConcurrency <= 0 {
		c.SubAgentDispatchConcurrency = defaults.SubAgentDispatchConcurrency
	}
}

// Validate 校验 runtime 配置是否满足最小约束。
func (c RuntimeConfig) Validate() error {
	if c.MaxNoProgressStreak <= 0 {
		return errors.New("max_no_progress_streak must be greater than 0")
	}
	if c.MaxRepeatCycleStreak <= 0 {
		return errors.New("max_repeat_cycle_streak must be greater than 0")
	}
	if c.MaxTurn <= 0 {
		return errors.New("max_turn must be greater than 0")
	}
	if c.SubAgentDispatchConcurrency <= 0 {
		return errors.New("subagent_dispatch_concurrency must be greater than 0")
	}
	return nil
}
