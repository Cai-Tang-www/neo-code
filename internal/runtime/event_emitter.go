package runtime

import (
	"context"
	"sync/atomic"
	"time"

	"neo-code/internal/runtime/controlplane"
)

const turnUnspecified = -1

// emit 将 runtime 事件投递到事件通道，并在通道阻塞且上下文取消时返回错误。
func (s *Service) emit(ctx context.Context, kind EventType, runID string, sessionID string, payload any) error {
	return s.emitWithEnvelope(ctx, RuntimeEvent{
		Type:           kind,
		RunID:          runID,
		SessionID:      sessionID,
		Turn:           turnUnspecified,
		Timestamp:      time.Now(),
		PayloadVersion: controlplane.PayloadVersion,
		Payload:        payload,
	})
}

// emitRunScoped 携带当前 run 的 turn/phase 元数据发出事件。
func (s *Service) emitRunScoped(ctx context.Context, kind EventType, state *runState, payload any) error {
	if state == nil {
		return s.emit(ctx, kind, "", "", payload)
	}
	phase := ""
	if state.phase != "" {
		phase = string(state.phase)
	}
	return s.emitWithEnvelope(ctx, RuntimeEvent{
		Type:           kind,
		RunID:          state.runID,
		SessionID:      state.session.ID,
		Turn:           state.turn,
		Phase:          phase,
		Timestamp:      time.Now(),
		PayloadVersion: controlplane.PayloadVersion,
		Payload:        payload,
	})
}

func (s *Service) emitWithEnvelope(ctx context.Context, evt RuntimeEvent) error {
	if evt.PayloadVersion == 0 {
		evt.PayloadVersion = controlplane.PayloadVersion
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	if evt.Sequence == 0 {
		evt.Sequence = s.nextEventSequence()
	}
	if err := s.deliverEvent(ctx, evt); err != nil {
		return err
	}
	return nil
}

// nextEventSequence 生成单个 Service 实例内严格递增的事件序号，便于消费端做乱序/重复校验。
func (s *Service) nextEventSequence() uint64 {
	if s == nil {
		return 0
	}
	return atomic.AddUint64(&s.eventSequence, 1)
}

func (s *Service) deliverEvent(ctx context.Context, evt RuntimeEvent) error {
	select {
	case s.events <- evt:
		return nil
	default:
	}
	select {
	case s.events <- evt:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
