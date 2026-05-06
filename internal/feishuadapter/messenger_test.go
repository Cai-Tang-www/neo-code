package feishuadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type queuedHTTPResponse struct {
	status int
	body   string
	err    error
}

type queuedHTTPClient struct {
	mu        sync.Mutex
	responses []queuedHTTPResponse
	requests  []*http.Request
	bodies    [][]byte
}

func (c *queuedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if req != nil {
		clone := req.Clone(req.Context())
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			c.bodies = append(c.bodies, body)
			req.Body = io.NopCloser(bytes.NewReader(body))
			clone.Body = io.NopCloser(bytes.NewReader(body))
		}
		c.requests = append(c.requests, clone)
	}
	if len(c.responses) == 0 {
		return nil, assertErr("unexpected http call")
	}
	current := c.responses[0]
	c.responses = c.responses[1:]
	if current.err != nil {
		return nil, current.err
	}
	return &http.Response{
		StatusCode: current.status,
		Body:       io.NopCloser(strings.NewReader(current.body)),
		Header:     make(http.Header),
	}, nil
}

func TestSendMessageRequiresFeishuBusinessCodeZero(t *testing.T) {
	client := &queuedHTTPClient{
		responses: []queuedHTTPResponse{
			{
				status: 200,
				body:   `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`,
			},
			{
				status: 200,
				body:   `{"code":999,"msg":"forbidden"}`,
			},
		},
	}
	messenger := NewFeishuMessenger("app", "secret", client)
	err := messenger.SendText(context.Background(), "chat-id", "hello")
	if err == nil {
		t.Fatal("expected send message business error")
	}
	if !strings.Contains(err.Error(), "code=999") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendMessageSuccessWhenHTTPAndBusinessCodePass(t *testing.T) {
	client := &queuedHTTPClient{
		responses: []queuedHTTPResponse{
			{
				status: 200,
				body:   `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`,
			},
			{
				status: 200,
				body:   `{"code":0,"msg":"ok","data":{"message_id":"mid"}}`,
			},
		},
	}
	messenger := NewFeishuMessenger("app", "secret", client)
	if err := messenger.SendText(context.Background(), "chat-id", "hello"); err != nil {
		t.Fatalf("send message: %v", err)
	}
}

func TestSendPermissionCardAndStatusCardUseCachedToken(t *testing.T) {
	client := &queuedHTTPClient{
		responses: []queuedHTTPResponse{
			{status: 200, body: `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`},
			{status: 200, body: `{"code":0,"msg":"ok","data":{"message_id":"permission-mid"}}`},
			{status: 200, body: `{"code":0,"msg":"ok","data":{"message_id":"status-mid"}}`},
		},
	}
	messenger := NewFeishuMessenger("app", "secret", client)
	if err := messenger.SendPermissionCard(context.Background(), "chat-id", PermissionCardPayload{
		RequestID: "perm-1",
		Message:   "请审批",
	}); err != nil {
		t.Fatalf("send permission card: %v", err)
	}
	cardID, err := messenger.SendStatusCard(context.Background(), "chat-id", StatusCardPayload{
		TaskName:        "生成周报",
		Status:          "planning",
		ApprovalStatus:  "pending",
		Result:          "pending",
		Summary:         "摘要",
		AsyncRewakeHint: "async_rewake",
	})
	if err != nil {
		t.Fatalf("send status card: %v", err)
	}
	if cardID != "status-mid" {
		t.Fatalf("card id = %q, want status-mid", cardID)
	}
	if len(client.requests) != 3 {
		t.Fatalf("request count = %d, want 3", len(client.requests))
	}
	if got := client.requests[0].URL.Path; got != "/open-apis/auth/v3/tenant_access_token/internal" {
		t.Fatalf("token path = %q", got)
	}
	for _, index := range []int{1, 2} {
		if authorization := client.requests[index].Header.Get("Authorization"); authorization != "Bearer token" {
			t.Fatalf("request %d auth header = %q", index, authorization)
		}
	}
	if len(client.bodies) != 3 {
		t.Fatalf("captured bodies = %d, want 3", len(client.bodies))
	}
	if !strings.Contains(string(client.bodies[1]), `perm-1`) {
		t.Fatalf("permission body = %s", string(client.bodies[1]))
	}
	if !strings.Contains(string(client.bodies[2]), "NeoCode 任务状态") || !strings.Contains(string(client.bodies[2]), "回灌：async_rewake") {
		t.Fatalf("status body = %s", string(client.bodies[2]))
	}
}

func TestUpdateCardCoversPatchRequestAndFallbacks(t *testing.T) {
	client := &queuedHTTPClient{
		responses: []queuedHTTPResponse{
			{status: 200, body: `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`},
			{status: 200, body: `{"code":0,"msg":"ok","data":{"message_id":"updated"}}`},
		},
	}
	messenger := NewFeishuMessenger("app", "secret", client)
	if err := messenger.UpdateCard(context.Background(), "om_card_1", StatusCardPayload{}); err != nil {
		t.Fatalf("update card: %v", err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(client.requests))
	}
	update := client.requests[1]
	if update.Method != http.MethodPatch || update.URL.Path != "/open-apis/im/v1/messages/om_card_1" {
		t.Fatalf("unexpected update request: method=%s path=%s", update.Method, update.URL.Path)
	}
	if !strings.Contains(string(client.bodies[1]), "任务：未命名任务") || !strings.Contains(string(client.bodies[1]), "结果：pending") {
		t.Fatalf("update body = %s", string(client.bodies[1]))
	}
}

func TestDoJSONRequestWithMessageIDCoversHTTPAndDecodeFailures(t *testing.T) {
	testCases := []struct {
		name string
		resp queuedHTTPResponse
		want string
	}{
		{name: "http error", resp: queuedHTTPResponse{err: assertErr("dial failed")}, want: "dial failed"},
		{name: "non 2xx invalid json", resp: queuedHTTPResponse{status: 500, body: "not-json"}, want: "status=500 body=invalid_json"},
		{name: "success invalid json", resp: queuedHTTPResponse{status: 200, body: "not-json"}, want: "invalid response body"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &queuedHTTPClient{responses: []queuedHTTPResponse{testCase.resp}}
			messenger := &feishuMessenger{httpClient: client}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			_, err = messenger.doJSONRequestWithMessageID(req)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want substring %q", err, testCase.want)
			}
		})
	}
}

func TestTenantAccessTokenCachesAndRecoversExpireFallback(t *testing.T) {
	client := &queuedHTTPClient{
		responses: []queuedHTTPResponse{
			{status: 200, body: `{"code":0,"msg":"ok","tenant_access_token":"token","expire":0}`},
		},
	}
	messenger := NewFeishuMessenger(" app ", " secret ", client).(*feishuMessenger)
	tokenA, err := messenger.tenantAccessToken(context.Background())
	if err != nil {
		t.Fatalf("tenant token first: %v", err)
	}
	tokenB, err := messenger.tenantAccessToken(context.Background())
	if err != nil {
		t.Fatalf("tenant token second: %v", err)
	}
	if tokenA != "token" || tokenB != "token" {
		t.Fatalf("unexpected tokens: %q %q", tokenA, tokenB)
	}
	if len(client.requests) != 1 {
		t.Fatalf("request count = %d, want cached single fetch", len(client.requests))
	}
	if messenger.appID != "app" || messenger.appSecret != "secret" {
		t.Fatalf("credentials not trimmed: %q %q", messenger.appID, messenger.appSecret)
	}
}

func TestBuildStatusCardAndFallbackStatusField(t *testing.T) {
	card := buildStatusCard(StatusCardPayload{
		TaskName:        "整理日报",
		Status:          "running",
		ApprovalStatus:  "approved",
		Result:          "success",
		Summary:         "已完成",
		AsyncRewakeHint: "hook_notification",
	})
	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal card: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "整理日报") || !strings.Contains(text, "摘要：已完成") || !strings.Contains(text, "回灌：hook_notification") {
		t.Fatalf("card payload = %s", text)
	}
	if fallbackStatusField("  ", "thinking") != "thinking" {
		t.Fatal("expected fallback for blank value")
	}
	if fallbackStatusField(" running ", "thinking") != "running" {
		t.Fatal("expected trimmed non-empty value")
	}
}
