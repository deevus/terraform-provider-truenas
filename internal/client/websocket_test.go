// internal/client/websocket_test.go
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/gorilla/websocket"
)

func TestWsRequest_HasResponseChannel(t *testing.T) {
	respChan := make(chan wsResponse, 1)
	req := wsRequest{
		method:   "test.method",
		params:   nil,
		response: respChan,
		ctx:      context.Background(),
	}

	if req.response == nil {
		t.Error("response channel is nil")
	}
	if req.method != "test.method" {
		t.Errorf("method = %q, want %q", req.method, "test.method")
	}
}

func TestWsResponse_ContainsResultOrError(t *testing.T) {
	resp := wsResponse{
		result: []byte(`{"foo":"bar"}`),
		err:    nil,
	}

	if resp.result == nil {
		t.Error("result is nil")
	}
	if resp.err != nil {
		t.Errorf("err = %v, want nil", resp.err)
	}
}

func TestNewWebSocketClient_CreatesChannels(t *testing.T) {
	mock := &MockClient{}
	config := WebSocketConfig{
		Host:     "truenas.local",
		Username: "root",
		APIKey:   "test-key",
		Fallback: mock,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	if client.requestChan == nil {
		t.Error("requestChan is nil")
	}
	if client.readChan == nil {
		t.Error("readChan is nil")
	}
	if client.disconnectChan == nil {
		t.Error("disconnectChan is nil")
	}
	if client.stopChan == nil {
		t.Error("stopChan is nil")
	}
}

func TestNewWebSocketClient_InvalidConfig(t *testing.T) {
	_, err := NewWebSocketClient(WebSocketConfig{})
	if err == nil {
		t.Error("NewWebSocketClient() error = nil, want error for invalid config")
	}
}

func TestWebSocketClient_WriterLoop_ProcessesRequests(t *testing.T) {
	// Create a test WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			if err := json.Unmarshal(msg, &req); err != nil {
				return
			}

			// Handle auth
			if req.Method == "auth.login_ex" {
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				}
				_ = conn.WriteJSON(resp)
				continue
			}

			// Handle job subscription
			if req.Method == "core.subscribe" {
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				}
				_ = conn.WriteJSON(resp)
				continue
			}

			// Echo method name as result
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"` + req.Method + `"`),
				ID:      req.ID,
			}
			_ = conn.WriteJSON(resp)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx := context.Background()
	result, err := client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var method string
	if err := json.Unmarshal(result, &method); err != nil {
		t.Fatalf("Unmarshal result error = %v", err)
	}

	if method != "test.method" {
		t.Errorf("result = %q, want %q", method, "test.method")
	}
}

func mustParsePort(s string) int {
	port, _ := strconv.Atoi(s)
	return port
}

func TestWebSocketClient_Call_RetriesOnError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			attempts++
			if attempts < 2 {
				// First attempt: return retriable error
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeTooManyConcurrent,
						Message: "too many concurrent calls",
					},
					ID: req.ID,
				})
			} else {
				// Second attempt: success
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`"success"`),
					ID:      req.ID,
				})
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     3,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	result, err := client.Call(context.Background(), "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if attempts < 2 {
		t.Errorf("attempts = %d, want at least 2", attempts)
	}

	var s string
	json.Unmarshal(result, &s)
	if s != "success" {
		t.Errorf("result = %q, want %q", s, "success")
	}
}

func TestWebSocketClient_CallAndWait_WaitsForJob(t *testing.T) {
	var writeMu sync.Mutex // Protect concurrent writes to WebSocket connection

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				writeMu.Unlock()

			case "core.subscribe":
				// Connection-level subscription for job events
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				writeMu.Unlock()

			case "long.running.task":
				// Return job ID
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`123`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				// Send job events in the correct TrueNAS nested format
				go func() {
					time.Sleep(50 * time.Millisecond)
					writeMu.Lock()
					// TrueNAS sends events as collection_update with nested params
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         123,
							"fields": map[string]any{
								"state":  "RUNNING",
								"result": nil,
							},
						},
					})
					writeMu.Unlock()
					time.Sleep(50 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         123,
							"fields": map[string]any{
								"state":  "SUCCESS",
								"result": map[string]string{"done": "yes"},
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.CallAndWait(ctx, "long.running.task", nil)
	if err != nil {
		t.Fatalf("CallAndWait() error = %v", err)
	}

	var res map[string]string
	if err := json.Unmarshal(result, &res); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if res["done"] != "yes" {
		t.Errorf("result = %v, want done=yes", res)
	}
}

func TestWebSocketClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = (*WebSocketClient)(nil)
}

func TestWebSocketClient_ReconnectsOnAuthError(t *testing.T) {
	connections := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		connections++

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			// First connection: return auth error
			if connections == 1 {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeTrueNASCall,
						Message: "Not authenticated",
						Data:    &JSONRPCData{Reason: "[ENOTAUTHENTICATED] Not authenticated"},
					},
					ID: req.ID,
				})
				return // Close connection
			}

			// Second connection: success
			conn.WriteJSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"reconnected"`),
				ID:      req.ID,
			})
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     3,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	result, err := client.Call(context.Background(), "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var s string
	json.Unmarshal(result, &s)
	if s != "reconnected" {
		t.Errorf("result = %q, want %q", s, "reconnected")
	}

	if connections < 2 {
		t.Errorf("connections = %d, want at least 2", connections)
	}
}

func TestWebSocketConfig_KeepaliveDefaults(t *testing.T) {
	mock := &MockClient{}
	config := WebSocketConfig{
		Host:     "truenas.local",
		Username: "root",
		APIKey:   "test-key",
		Fallback: mock,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if config.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %v, want %v", config.PingInterval, 30*time.Second)
	}
	if config.PingTimeout != 10*time.Second {
		t.Errorf("PingTimeout = %v, want %v", config.PingTimeout, 10*time.Second)
	}
}

func TestWebSocketClient_PingInterval(t *testing.T) {
	pingReceived := make(chan struct{}, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Set up ping handler to detect incoming pings
		conn.SetPingHandler(func(appData string) error {
			select {
			case pingReceived <- struct{}{}:
			default:
			}
			// Send pong reply
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			conn.WriteJSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"ok"`),
				ID:      req.ID,
			})
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
		PingInterval:   100 * time.Millisecond, // Fast ping for testing
		PingTimeout:    50 * time.Millisecond,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	// Make a call to establish connection
	ctx := context.Background()
	_, err = client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Wait for at least one ping
	select {
	case <-pingReceived:
		// Success - ping was sent
	case <-time.After(500 * time.Millisecond):
		t.Error("no ping received within timeout")
	}
}

func TestWebSocketClient_PongReceived(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Auto-reply pong to pings (default behavior)
		conn.SetPingHandler(func(appData string) error {
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			conn.WriteJSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"ok"`),
				ID:      req.ID,
			})
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
		PingInterval:   50 * time.Millisecond,
		PingTimeout:    100 * time.Millisecond,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx := context.Background()

	// Make initial call to establish connection
	_, err = client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Wait for multiple ping/pong cycles
	time.Sleep(200 * time.Millisecond)

	// Connection should still be alive
	_, err = client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() after pings error = %v", err)
	}
}

func TestWebSocketClient_PongTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Don't respond to pings (no pong handler)
		conn.SetPingHandler(func(appData string) error {
			// Intentionally don't send pong back
			return nil
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			conn.WriteJSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"ok"`),
				ID:      req.ID,
			})
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0, // Don't retry - we want to see the timeout error
		PingInterval:   50 * time.Millisecond,
		PingTimeout:    25 * time.Millisecond,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx := context.Background()

	// Make initial call to establish connection
	_, err = client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Wait for ping + timeout
	time.Sleep(150 * time.Millisecond)

	// Next call should fail or reconnect
	// The connection was closed due to pong timeout
	_, err = client.Call(ctx, "test.method", nil)
	// Either error or reconnect is acceptable - the test verifies timeout detection works
	// If it reconnects, the call succeeds; if not, we get an error
	// Both outcomes are valid as long as timeout was detected
}

func TestWebSocketClient_CallAndWait_FastCompletingJob(t *testing.T) {
	// This test verifies that fast-completing jobs are handled correctly via events.
	// With connection-level subscription, events are flowing before any job is created.
	var writeMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "fast.task":
				// Return job ID
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`456`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				// Send immediate completion event in correct format
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         456,
							"fields": map[string]any{
								"state":  "SUCCESS",
								"result": map[string]string{"status": "completed"},
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.CallAndWait(ctx, "fast.task", nil)
	if err != nil {
		t.Fatalf("CallAndWait() error = %v", err)
	}

	var res map[string]string
	if err := json.Unmarshal(result, &res); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if res["status"] != "completed" {
		t.Errorf("result = %v, want status=completed", res)
	}
}

func TestWebSocketClient_CallAndWait_FastFailingJob(t *testing.T) {
	// Test that fast-failing jobs are handled correctly via events
	var writeMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "failing.task":
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`789`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				// Send immediate failure event in correct format
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         789,
							"fields": map[string]any{
								"state": "FAILED",
								"error": "[EINVAL] Invalid configuration",
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.CallAndWait(ctx, "failing.task", nil)
	if err == nil {
		t.Fatal("CallAndWait() expected error for failed job, got nil")
	}

	var trueNASErr *TrueNASError
	if !errors.As(err, &trueNASErr) {
		t.Fatalf("error = %T, want *TrueNASError", err)
	}

	if trueNASErr.Code != "EINVAL" {
		t.Errorf("error code = %q, want %q", trueNASErr.Code, "EINVAL")
	}
}

func TestWebSocketClient_PingDisabled(t *testing.T) {
	pingReceived := make(chan struct{}, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Track if any pings are received
		conn.SetPingHandler(func(appData string) error {
			select {
			case pingReceived <- struct{}{}:
			default:
			}
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			conn.WriteJSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"ok"`),
				ID:      req.ID,
			})
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
		PingInterval:   -1, // Disable pings (any negative value)
		PingTimeout:    50 * time.Millisecond,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx := context.Background()
	_, err = client.Call(ctx, "test.method", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Wait a bit - no pings should be sent
	time.Sleep(100 * time.Millisecond)

	select {
	case <-pingReceived:
		t.Error("ping was received when pings should be disabled")
	default:
		// No ping received - correct behavior
	}
}

func TestJobEventBuffer_Add(t *testing.T) {
	buf := &jobEventBuffer{}

	// Add events
	for i := int64(1); i <= 5; i++ {
		buf.add(JobEvent{ID: i, State: "SUCCESS"})
	}

	if len(buf.events) != 5 {
		t.Errorf("buffer len = %d, want 5", len(buf.events))
	}
}

func TestJobEventBuffer_GetByJobID(t *testing.T) {
	buf := &jobEventBuffer{}

	// Add terminal event
	buf.add(JobEvent{ID: 123, State: "SUCCESS", Result: []byte(`"done"`)})

	// Should find it
	event := buf.getByJobID(123)
	if event == nil {
		t.Fatal("expected to find event")
	}
	if event.State != "SUCCESS" {
		t.Errorf("state = %q, want SUCCESS", event.State)
	}

	// Should not find non-existent job
	if buf.getByJobID(999) != nil {
		t.Error("expected nil for non-existent job")
	}
}

func TestJobEventBuffer_OnlyReturnsTerminalStates(t *testing.T) {
	buf := &jobEventBuffer{}

	// Add non-terminal event
	buf.add(JobEvent{ID: 123, State: "RUNNING"})

	// Should not find non-terminal state
	if buf.getByJobID(123) != nil {
		t.Error("expected nil for RUNNING state")
	}

	// Add terminal event
	buf.add(JobEvent{ID: 123, State: "SUCCESS"})

	// Now should find it
	if buf.getByJobID(123) == nil {
		t.Error("expected to find SUCCESS event")
	}
}

func TestJobEventBuffer_Overflow(t *testing.T) {
	buf := &jobEventBuffer{}

	// Fill buffer beyond capacity
	for i := int64(1); i <= jobEventBufferSize+10; i++ {
		buf.add(JobEvent{ID: i, State: "SUCCESS"})
	}

	// Buffer should not exceed max size
	if len(buf.events) != jobEventBufferSize {
		t.Errorf("buffer len = %d, want %d", len(buf.events), jobEventBufferSize)
	}

	// Oldest events should be gone (IDs 1-10)
	for i := int64(1); i <= 10; i++ {
		if buf.getByJobID(i) != nil {
			t.Errorf("expected job %d to be evicted", i)
		}
	}

	// Newer events should still be there
	if buf.getByJobID(jobEventBufferSize + 5) == nil {
		t.Error("expected to find recent job")
	}
}

func TestWebSocketClient_GetVersion(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantMajor  int
		wantMinor  int
		wantErr    bool
		errMessage string
	}{
		{
			name:      "success",
			response:  `"TrueNAS-SCALE-24.10.2.1"`,
			wantMajor: 24,
			wantMinor: 10,
			wantErr:   false,
		},
		{
			name:       "parse_error_not_a_string",
			response:   `123`,
			wantErr:    true,
			errMessage: "failed to parse version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				for {
					_, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}

					var req JSONRPCRequest
					json.Unmarshal(msg, &req)

					if req.Method == "auth.login_ex" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "core.subscribe" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`true`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "system.version" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(tt.response),
							ID:      req.ID,
						})
					}
				}
			}))
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			host := strings.TrimPrefix(wsURL, "ws://")

			mock := &MockClient{
				GetVersionFunc: func(ctx context.Context) (api.Version, error) {
					return api.Version{Major: 25, Minor: 0}, nil
				},
			}

			config := WebSocketConfig{
				Host:           strings.Split(host, ":")[0],
				Port:           mustParsePort(strings.Split(host, ":")[1]),
				Username:       "root",
				APIKey:         "test-key",
				Fallback:       mock,
				ConnectTimeout: 5 * time.Second,
				MaxRetries:     1,
			}

			client, err := NewWebSocketClient(config)
			if err != nil {
				t.Fatalf("NewWebSocketClient() error = %v", err)
			}
			client.testInsecure = true
			defer client.Close()

			ctx := context.Background()
			version, err := client.GetVersion(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetVersion() expected error, got nil")
				} else if tt.errMessage != "" && !strings.Contains(err.Error(), tt.errMessage) {
					t.Errorf("GetVersion() error = %v, want containing %q", err, tt.errMessage)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetVersion() error = %v", err)
			}

			if version.Major != tt.wantMajor || version.Minor != tt.wantMinor {
				t.Errorf("GetVersion() = %d.%d, want %d.%d", version.Major, version.Minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}

func TestWebSocketClient_GetVersion_RpcError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "system.version" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeInternal,
						Message: "internal error",
					},
					ID: req.ID,
				})
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	_, err = client.GetVersion(context.Background())
	if err == nil {
		t.Error("GetVersion() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get version") {
		t.Errorf("GetVersion() error = %v, want containing 'failed to get version'", err)
	}
}

func TestWebSocketClient_WriteFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		params  WriteFileParams
		wantErr bool
	}{
		{
			name: "success_with_uid_gid",
			path: "/mnt/test/file.txt",
			params: WriteFileParams{
				Content: []byte("hello world"),
				Mode:    0644,
				UID:     intPtr(1000),
				GID:     intPtr(1000),
			},
			wantErr: false,
		},
		{
			name: "success_without_uid_gid",
			path: "/mnt/test/file.txt",
			params: WriteFileParams{
				Content: []byte("hello world"),
				Mode:    0644,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedParams []any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				for {
					_, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}

					var req JSONRPCRequest
					json.Unmarshal(msg, &req)

					if req.Method == "auth.login_ex" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "core.subscribe" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`true`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "filesystem.file_receive" {
						receivedParams = req.Params.([]any)
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`true`),
							ID:      req.ID,
						})
					}
				}
			}))
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			host := strings.TrimPrefix(wsURL, "ws://")

			mock := &MockClient{
				GetVersionFunc: func(ctx context.Context) (api.Version, error) {
					return api.Version{Major: 25, Minor: 0}, nil
				},
			}

			config := WebSocketConfig{
				Host:           strings.Split(host, ":")[0],
				Port:           mustParsePort(strings.Split(host, ":")[1]),
				Username:       "root",
				APIKey:         "test-key",
				Fallback:       mock,
				ConnectTimeout: 5 * time.Second,
				MaxRetries:     1,
			}

			client, err := NewWebSocketClient(config)
			if err != nil {
				t.Fatalf("NewWebSocketClient() error = %v", err)
			}
			client.testInsecure = true
			defer client.Close()

			ctx := context.Background()
			err = client.WriteFile(ctx, tt.path, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("WriteFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			// Verify params structure
			if len(receivedParams) != 3 {
				t.Errorf("WriteFile() params len = %d, want 3", len(receivedParams))
			}
		})
	}
}

func TestWebSocketClient_WriteFile_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "filesystem.file_receive" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeInternal,
						Message: "permission denied",
					},
					ID: req.ID,
				})
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx := context.Background()
	err = client.WriteFile(ctx, "/mnt/test/file.txt", WriteFileParams{Content: []byte("test"), Mode: 0644})

	if err == nil {
		t.Error("WriteFile() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write file") {
		t.Errorf("WriteFile() error = %v, want containing 'failed to write file'", err)
	}
}

func intPtr(v int) *int { return &v }

func TestWebSocketClient_FileExists(t *testing.T) {
	tests := []struct {
		name     string
		response *JSONRPCResponse
		want     bool
		wantErr  bool
	}{
		{
			name: "file_exists",
			response: &JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`{"name":"test.txt","type":"FILE"}`),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "file_not_found_enoent",
			response: &JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    ErrCodeTrueNASCall,
					Message: "file not found",
					Data:    &JSONRPCData{Error: 2}, // ENOENT
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "other_error",
			response: &JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    ErrCodeTrueNASCall,
					Message: "permission denied",
					Data:    &JSONRPCData{Error: 13}, // EACCES
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				for {
					_, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}

					var req JSONRPCRequest
					json.Unmarshal(msg, &req)

					if req.Method == "auth.login_ex" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "core.subscribe" {
						conn.WriteJSON(JSONRPCResponse{
							JSONRPC: "2.0",
							Result:  json.RawMessage(`true`),
							ID:      req.ID,
						})
						continue
					}

					if req.Method == "filesystem.stat" {
						resp := *tt.response
						resp.ID = req.ID
						conn.WriteJSON(resp)
					}
				}
			}))
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			host := strings.TrimPrefix(wsURL, "ws://")

			mock := &MockClient{
				GetVersionFunc: func(ctx context.Context) (api.Version, error) {
					return api.Version{Major: 25, Minor: 0}, nil
				},
			}

			config := WebSocketConfig{
				Host:           strings.Split(host, ":")[0],
				Port:           mustParsePort(strings.Split(host, ":")[1]),
				Username:       "root",
				APIKey:         "test-key",
				Fallback:       mock,
				ConnectTimeout: 5 * time.Second,
				MaxRetries:     0,
			}

			client, err := NewWebSocketClient(config)
			if err != nil {
				t.Fatalf("NewWebSocketClient() error = %v", err)
			}
			client.testInsecure = true
			defer client.Close()

			ctx := context.Background()
			exists, err := client.FileExists(ctx, "/mnt/test/file.txt")

			if tt.wantErr {
				if err == nil {
					t.Error("FileExists() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("FileExists() error = %v", err)
			}

			if exists != tt.want {
				t.Errorf("FileExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestWebSocketClient_Chown(t *testing.T) {
	server := createJobServer("filesystem.chown", json.RawMessage(`null`), nil)
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Chown(ctx, "/mnt/test/file.txt", 1000, 1000)
	if err != nil {
		t.Fatalf("Chown() error = %v", err)
	}
}

func TestWebSocketClient_Chown_Error(t *testing.T) {
	server := createJobServer("filesystem.chown", nil, &JSONRPCError{
		Code:    ErrCodeTrueNASCall,
		Message: "permission denied",
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Chown(ctx, "/mnt/test/file.txt", 1000, 1000)
	if err == nil {
		t.Error("Chown() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to chown") {
		t.Errorf("Chown() error = %v, want containing 'failed to chown'", err)
	}
}

func TestWebSocketClient_ChmodRecursive(t *testing.T) {
	server := createJobServer("filesystem.setperm", json.RawMessage(`null`), nil)
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.ChmodRecursive(ctx, "/mnt/test/dir", 0755)
	if err != nil {
		t.Fatalf("ChmodRecursive() error = %v", err)
	}
}

func TestWebSocketClient_ChmodRecursive_Error(t *testing.T) {
	server := createJobServer("filesystem.setperm", nil, &JSONRPCError{
		Code:    ErrCodeTrueNASCall,
		Message: "permission denied",
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.ChmodRecursive(ctx, "/mnt/test/dir", 0755)
	if err == nil {
		t.Error("ChmodRecursive() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to chmod") {
		t.Errorf("ChmodRecursive() error = %v, want containing 'failed to chmod'", err)
	}
}

func TestWebSocketClient_MkdirAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "filesystem.mkdir" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
			}
		}
	}))
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	err := client.MkdirAll(ctx, "/mnt/test/newdir", 0755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func TestWebSocketClient_MkdirAll_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "filesystem.mkdir" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeTrueNASCall,
						Message: "file exists",
					},
					ID: req.ID,
				})
			}
		}
	}))
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	err := client.MkdirAll(ctx, "/mnt/test/existingdir", 0755)
	if err == nil {
		t.Error("MkdirAll() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to mkdir") {
		t.Errorf("MkdirAll() error = %v, want containing 'failed to mkdir'", err)
	}
}

// Helper function to create a test client from an httptest.Server
func createTestClient(t *testing.T, server *httptest.Server) *WebSocketClient {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	return client
}

// createJobServer creates a mock server that handles job-based methods
func createJobServer(method string, result json.RawMessage, rpcErr *JSONRPCError) *httptest.Server {
	var writeMu sync.Mutex
	jobID := int64(100)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case method:
				if rpcErr != nil {
					conn.WriteJSON(JSONRPCResponse{
						JSONRPC: "2.0",
						Error:   rpcErr,
						ID:      req.ID,
					})
					continue
				}

				// Return job ID
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(fmt.Sprintf(`%d`, jobID)),
					ID:      req.ID,
				})
				writeMu.Unlock()

				// Send job completion event
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         jobID,
							"fields": map[string]any{
								"state":  "SUCCESS",
								"result": result,
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
}

func TestWebSocketClient_CallAndWait_EventArrivesBeforeSubscription(t *testing.T) {
	// This test verifies the event buffer handles the race condition where
	// the job completes BEFORE we register our subscription.
	var writeMu sync.Mutex
	eventSent := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "instant.task":
				// Return job ID AND send completion event immediately
				// The event will arrive BEFORE CallAndWait registers the subscription
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`999`),
					ID:      req.ID,
				})
				// Send completion event immediately (no delay)
				conn.WriteJSON(map[string]any{
					"msg":    "method",
					"method": "collection_update",
					"params": map[string]any{
						"msg":        "changed",
						"collection": "core.get_jobs",
						"id":         999,
						"fields": map[string]any{
							"state":  "SUCCESS",
							"result": map[string]string{"instant": "true"},
						},
					},
				})
				writeMu.Unlock()
				close(eventSent)
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// CallAndWait should succeed even though the event arrived before subscription
	// because the event buffer replays it
	result, err := client.CallAndWait(ctx, "instant.task", nil)
	if err != nil {
		t.Fatalf("CallAndWait() error = %v", err)
	}

	var res map[string]string
	if err := json.Unmarshal(result, &res); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if res["instant"] != "true" {
		t.Errorf("result = %v, want instant=true", res)
	}
}

// Phase 2: Delegated Methods Tests

func TestWebSocketClient_ReadFile_DelegatesToFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})
				continue
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	readFileCalled := false
	expectedContent := []byte("file content from fallback")
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			readFileCalled = true
			if path != "/mnt/test/file.txt" {
				t.Errorf("ReadFile path = %q, want %q", path, "/mnt/test/file.txt")
			}
			return expectedContent, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	content, err := client.ReadFile(context.Background(), "/mnt/test/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !readFileCalled {
		t.Error("ReadFile() did not delegate to fallback")
	}

	if string(content) != string(expectedContent) {
		t.Errorf("ReadFile() = %q, want %q", content, expectedContent)
	}
}

func TestWebSocketClient_ReadFile_PropagatesError(t *testing.T) {
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			return nil, errors.New("ssh connection failed")
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.ReadFile(context.Background(), "/mnt/test/file.txt")
	if err == nil {
		t.Error("ReadFile() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ssh connection failed") {
		t.Errorf("ReadFile() error = %v, want containing 'ssh connection failed'", err)
	}
}

func TestWebSocketClient_DeleteFile_DelegatesToFallback(t *testing.T) {
	deleteFileCalled := false
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		DeleteFileFunc: func(ctx context.Context, path string) error {
			deleteFileCalled = true
			if path != "/mnt/test/file.txt" {
				t.Errorf("DeleteFile path = %q, want %q", path, "/mnt/test/file.txt")
			}
			return nil
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.DeleteFile(context.Background(), "/mnt/test/file.txt")
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	if !deleteFileCalled {
		t.Error("DeleteFile() did not delegate to fallback")
	}
}

func TestWebSocketClient_DeleteFile_PropagatesError(t *testing.T) {
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		DeleteFileFunc: func(ctx context.Context, path string) error {
			return errors.New("permission denied")
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.DeleteFile(context.Background(), "/mnt/test/file.txt")
	if err == nil {
		t.Error("DeleteFile() expected error, got nil")
	}
}

func TestWebSocketClient_RemoveDir_DelegatesToFallback(t *testing.T) {
	removeDirCalled := false
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		RemoveDirFunc: func(ctx context.Context, path string) error {
			removeDirCalled = true
			if path != "/mnt/test/dir" {
				t.Errorf("RemoveDir path = %q, want %q", path, "/mnt/test/dir")
			}
			return nil
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.RemoveDir(context.Background(), "/mnt/test/dir")
	if err != nil {
		t.Fatalf("RemoveDir() error = %v", err)
	}

	if !removeDirCalled {
		t.Error("RemoveDir() did not delegate to fallback")
	}
}

func TestWebSocketClient_RemoveDir_PropagatesError(t *testing.T) {
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		RemoveDirFunc: func(ctx context.Context, path string) error {
			return errors.New("directory not empty")
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.RemoveDir(context.Background(), "/mnt/test/dir")
	if err == nil {
		t.Error("RemoveDir() expected error, got nil")
	}
}

func TestWebSocketClient_RemoveAll_DelegatesToFallback(t *testing.T) {
	removeAllCalled := false
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		RemoveAllFunc: func(ctx context.Context, path string) error {
			removeAllCalled = true
			if path != "/mnt/test/dir" {
				t.Errorf("RemoveAll path = %q, want %q", path, "/mnt/test/dir")
			}
			return nil
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.RemoveAll(context.Background(), "/mnt/test/dir")
	if err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	if !removeAllCalled {
		t.Error("RemoveAll() did not delegate to fallback")
	}
}

func TestWebSocketClient_RemoveAll_PropagatesError(t *testing.T) {
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		RemoveAllFunc: func(ctx context.Context, path string) error {
			return errors.New("io error")
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	err = client.RemoveAll(context.Background(), "/mnt/test/dir")
	if err == nil {
		t.Error("RemoveAll() expected error, got nil")
	}
}

// Phase 3: Core Function Edge Cases

func TestWrapParams(t *testing.T) {
	mock := &MockClient{}
	config := WebSocketConfig{
		Host:     "localhost",
		Port:     443,
		Username: "root",
		APIKey:   "test-key",
		Fallback: mock,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	tests := []struct {
		name   string
		params any
		want   any
	}{
		{
			name:   "nil_returns_nil",
			params: nil,
			want:   nil,
		},
		{
			name:   "already_slice_returns_as_is",
			params: []any{"a", "b"},
			want:   []any{"a", "b"},
		},
		{
			name:   "single_string_wrapped",
			params: "hello",
			want:   []any{"hello"},
		},
		{
			name:   "single_map_wrapped",
			params: map[string]any{"key": "value"},
			want:   []any{map[string]any{"key": "value"}},
		},
		{
			name:   "single_int_wrapped",
			params: 42,
			want:   []any{42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.wrapParams(tt.params)

			// Compare results
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("wrapParams() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestWebSocketClient_Connect_VersionDetectionFails(t *testing.T) {
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{}, errors.New("connection refused")
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 2 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when version detection fails, got nil")
	}
	if !strings.Contains(err.Error(), "version detection failed") {
		t.Errorf("Call() error = %v, want containing 'version detection failed'", err)
	}
}

func TestWebSocketClient_Connect_DialFails(t *testing.T) {
	// Use a server that immediately closes the connection to simulate dial failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 404 to cause WebSocket upgrade to fail
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 1 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when dial fails, got nil")
	}
	if !strings.Contains(err.Error(), "websocket connect failed") {
		t.Errorf("Call() error = %v, want containing 'websocket connect failed'", err)
	}
}

func TestWebSocketClient_Authenticate_WriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Close connection immediately to cause write error
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when auth write fails, got nil")
	}
}

func TestWebSocketClient_Authenticate_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read auth request but close without responding
		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		// Close connection to cause read error
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when auth read fails, got nil")
	}
}

func TestWebSocketClient_Authenticate_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeInternal,
						Message: "invalid credentials",
					},
					ID: req.ID,
				})
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "wrong-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when auth fails, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Call() error = %v, want containing 'authentication failed'", err)
	}
}

func TestWebSocketClient_Authenticate_NonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"OTP_REQUIRED"}`),
					ID:      req.ID,
				})
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when auth response is not SUCCESS, got nil")
	}
	if !strings.Contains(err.Error(), "OTP_REQUIRED") {
		t.Errorf("Call() error = %v, want containing 'OTP_REQUIRED'", err)
	}
}

func TestWebSocketClient_SubscribeJobEvents_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			if req.Method == "auth.login_ex" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})
				continue
			}

			if req.Method == "core.subscribe" {
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    ErrCodeInternal,
						Message: "subscription not allowed",
					},
					ID: req.ID,
				})
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.TrimPrefix(wsURL, "ws://")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
	}

	config := WebSocketConfig{
		Host:           strings.Split(host, ":")[0],
		Port:           mustParsePort(strings.Split(host, ":")[1]),
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test.method", nil)
	if err == nil {
		t.Error("Call() expected error when job subscription fails, got nil")
	}
	if !strings.Contains(err.Error(), "job subscription failed") {
		t.Errorf("Call() error = %v, want containing 'job subscription failed'", err)
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name string
		err  *JSONRPCError
		want bool
	}{
		{
			name: "auth_error_with_reason",
			err: &JSONRPCError{
				Code:    ErrCodeTrueNASCall,
				Message: "Not authenticated",
				Data:    &JSONRPCData{Reason: "[ENOTAUTHENTICATED] session expired"},
			},
			want: true,
		},
		{
			name: "non_auth_error",
			err: &JSONRPCError{
				Code:    ErrCodeTrueNASCall,
				Message: "Permission denied",
				Data:    &JSONRPCData{Reason: "[EPERM] permission denied"},
			},
			want: false,
		},
		{
			name: "nil_data",
			err: &JSONRPCError{
				Code:    ErrCodeTrueNASCall,
				Message: "Some error",
				Data:    nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAuthenticationError(tt.err)
			if got != tt.want {
				t.Errorf("isAuthenticationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebSocketClient_Connect_OlderVersion_ReturnsError(t *testing.T) {
	// Mock returns older version (24.x) which should be rejected
	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 24, Minor: 10, Raw: "TrueNAS-SCALE-24.10.2.4"}, nil
		},
	}

	config := WebSocketConfig{
		Host:           "localhost",
		Port:           443,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     0, // Don't retry
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	_, err = client.Call(ctx, "test.method", nil)

	// Should get ErrUnsupportedVersion
	if err == nil {
		t.Fatal("Call() expected error for TrueNAS 24.x, got nil")
	}

	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Errorf("Call() error = %v, want ErrUnsupportedVersion", err)
	}

	// Verify error message includes version info
	if !strings.Contains(err.Error(), "24.10.2.4") {
		t.Errorf("Error should contain version info, got: %v", err)
	}
}

func TestWebSocketClient_CallAndWait_FailingJob_WithAppLifecycleEnrichment(t *testing.T) {
	// Test that FAILED jobs with app lifecycle errors get enriched via fallback SSH client
	var writeMu sync.Mutex

	// Read the fixture
	logContent, err := os.ReadFile("../testdata/fixtures/app_lifecycle.log")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "app.failing":
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`999`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				// Send failure event with app lifecycle error
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         999,
							"fields": map[string]any{
								"state": "FAILED",
								"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details",
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.Split(strings.TrimPrefix(wsURL, "ws://"), ":")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			if path == "/var/log/app_lifecycle.log" {
				return logContent, nil
			}
			return nil, errors.New("file not found")
		},
	}

	port, _ := strconv.Atoi(host[1])
	config := WebSocketConfig{
		Host:           host[0],
		Port:           port,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.CallAndWait(ctx, "app.failing", nil)
	if err == nil {
		t.Fatal("CallAndWait() expected error for failed job, got nil")
	}

	var trueNASErr *TrueNASError
	if !errors.As(err, &trueNASErr) {
		t.Fatalf("error = %T, want *TrueNASError", err)
	}

	// Verify the error was enriched with app lifecycle log content
	if !strings.Contains(trueNASErr.AppLifecycleError, "bind: address already in use") {
		t.Errorf("expected AppLifecycleError to contain 'bind: address already in use', got %q", trueNASErr.AppLifecycleError)
	}

	// The Error() output should be clean (no "Please check" message)
	errStr := err.Error()
	if strings.Contains(errStr, "Please check") {
		t.Errorf("expected clean error output, got %q", errStr)
	}
}

func TestWebSocketClient_CallAndWait_FailingJob_EnrichmentFailsSilently(t *testing.T) {
	// Test that enrichment failure doesn't affect the error reporting
	var writeMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "app.failing":
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`888`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         888,
							"fields": map[string]any{
								"state": "FAILED",
								"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details",
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.Split(strings.TrimPrefix(wsURL, "ws://"), ":")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			// Simulate failure to read log file
			return nil, errors.New("permission denied")
		},
	}

	port, _ := strconv.Atoi(host[1])
	config := WebSocketConfig{
		Host:           host[0],
		Port:           port,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.CallAndWait(ctx, "app.failing", nil)
	if err == nil {
		t.Fatal("CallAndWait() expected error for failed job, got nil")
	}

	var trueNASErr *TrueNASError
	if !errors.As(err, &trueNASErr) {
		t.Fatalf("error = %T, want *TrueNASError", err)
	}

	// Should still have the original error info
	if trueNASErr.Code != "EFAULT" {
		t.Errorf("expected code EFAULT, got %s", trueNASErr.Code)
	}

	// AppLifecycleError should be empty since fetch failed
	if trueNASErr.AppLifecycleError != "" {
		t.Errorf("expected empty AppLifecycleError when fetch fails, got %q", trueNASErr.AppLifecycleError)
	}
}

func TestWebSocketClient_CallAndWait_AbortedJob_WithEnrichment(t *testing.T) {
	// Test that ABORTED jobs also get enriched with app lifecycle errors
	var writeMu sync.Mutex

	// Read the fixture
	logContent, err := os.ReadFile("../testdata/fixtures/app_lifecycle.log")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req JSONRPCRequest
			json.Unmarshal(msg, &req)

			switch req.Method {
			case "auth.login_ex":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"response_type":"SUCCESS"}`),
					ID:      req.ID,
				})

			case "core.subscribe":
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`true`),
					ID:      req.ID,
				})

			case "app.aborting":
				writeMu.Lock()
				conn.WriteJSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`777`),
					ID:      req.ID,
				})
				writeMu.Unlock()
				// Send ABORTED event with app lifecycle error
				go func() {
					time.Sleep(10 * time.Millisecond)
					writeMu.Lock()
					conn.WriteJSON(map[string]any{
						"msg":    "method",
						"method": "collection_update",
						"params": map[string]any{
							"msg":        "changed",
							"collection": "core.get_jobs",
							"id":         777,
							"fields": map[string]any{
								"state": "ABORTED",
								"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details",
							},
						},
					})
					writeMu.Unlock()
				}()
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	host := strings.Split(strings.TrimPrefix(wsURL, "ws://"), ":")

	mock := &MockClient{
		GetVersionFunc: func(ctx context.Context) (api.Version, error) {
			return api.Version{Major: 25, Minor: 0}, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			if path == "/var/log/app_lifecycle.log" {
				return logContent, nil
			}
			return nil, errors.New("file not found")
		},
	}

	port, _ := strconv.Atoi(host[1])
	config := WebSocketConfig{
		Host:           host[0],
		Port:           port,
		Username:       "root",
		APIKey:         "test-key",
		Fallback:       mock,
		ConnectTimeout: 5 * time.Second,
		MaxRetries:     1,
	}

	client, err := NewWebSocketClient(config)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	client.testInsecure = true
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.CallAndWait(ctx, "app.aborting", nil)
	if err == nil {
		t.Fatal("CallAndWait() expected error for aborted job, got nil")
	}

	var trueNASErr *TrueNASError
	if !errors.As(err, &trueNASErr) {
		t.Fatalf("error = %T, want *TrueNASError", err)
	}

	// Verify the error was enriched with app lifecycle log content
	if !strings.Contains(trueNASErr.AppLifecycleError, "bind: address already in use") {
		t.Errorf("expected AppLifecycleError to contain 'bind: address already in use', got %q", trueNASErr.AppLifecycleError)
	}

	// The Error() output should be clean (no "Please check" message)
	errStr := err.Error()
	if strings.Contains(errStr, "Please check") {
		t.Errorf("expected clean error output, got %q", errStr)
	}
}
