package feishuadapter

import (
	"encoding/json"
	"testing"
)

func TestCloneRawMessageCopiesBytes(t *testing.T) {
	original := json.RawMessage(`{"ok":true}`)
	cloned := cloneRawMessage(original)
	if string(cloned) != string(original) {
		t.Fatalf("clone = %s, want %s", string(cloned), string(original))
	}
	cloned[0] = '['
	if original[0] != '{' {
		t.Fatalf("expected original to remain unchanged, got %s", string(original))
	}
}

func TestParseGatewayRuntimeEventReadsEnvelopePayload(t *testing.T) {
	eventType, sessionID, runID, envelope, err := parseGatewayRuntimeEvent(json.RawMessage(`{
		"session_id":"session-1",
		"run_id":"run-1",
		"payload":{"event_type":"run_done","payload":{"content":"done"}}
	}`))
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}
	if eventType != "run_done" || sessionID != "session-1" || runID != "run-1" {
		t.Fatalf("unexpected frame fields: type=%q session=%q run=%q", eventType, sessionID, runID)
	}
	if readString(envelope, "content") != "done" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}
