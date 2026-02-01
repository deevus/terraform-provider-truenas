// internal/client/websocket.go
package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/gorilla/websocket"
)

// ErrCodeInternal is the error code for internal errors.
const ErrCodeInternal = -1

// WebSocketConfig contains configuration for the WebSocket client.
type WebSocketConfig struct {
	Host               string
	Username           string
	APIKey             string
	Port               int
	InsecureSkipVerify bool
	MaxConcurrent      int
	ConnectTimeout     time.Duration
	MaxRetries         int
	PingInterval       time.Duration // Interval between pings (0 = disabled, default: 30s)
	PingTimeout        time.Duration // Time to wait for pong (default: 10s)
	Fallback           Client        // Required for SSH-only operations
}

// Validate validates the WebSocketConfig and sets defaults.
func (c *WebSocketConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.APIKey == "" {
		return errors.New("api_key is required")
	}
	if c.Fallback == nil {
		return errors.New("fallback client is required")
	}

	// Set defaults
	if c.Port == 0 {
		c.Port = 443
	}
	if c.MaxConcurrent == 0 {
		c.MaxConcurrent = 20
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.PingInterval == 0 {
		c.PingInterval = 30 * time.Second
	}
	if c.PingTimeout == 0 {
		c.PingTimeout = 10 * time.Second
	}

	return nil
}

// Compile-time check that WebSocketClient implements Client.
var _ Client = (*WebSocketClient)(nil)

// wsRequest is sent from callers to the writer goroutine.
type wsRequest struct {
	method   string
	params   any
	response chan<- wsResponse
	ctx      context.Context
}

// wsResponse is sent back to callers.
type wsResponse struct {
	result json.RawMessage
	err    error
}

// wsSubscription for job events.
type wsSubscription struct {
	jobID int64
	ch    chan<- JobEvent
}

// jobEventBuffer maintains recent events for replay to new subscribers.
const jobEventBufferSize = 100

type jobEventBuffer struct {
	events []JobEvent
}

func (b *jobEventBuffer) add(event JobEvent) {
	if len(b.events) >= jobEventBufferSize {
		// Remove oldest event
		b.events = b.events[1:]
	}
	b.events = append(b.events, event)
}

func (b *jobEventBuffer) getByJobID(jobID int64) *JobEvent {
	// Search from newest to oldest for terminal states
	for i := len(b.events) - 1; i >= 0; i-- {
		if b.events[i].ID == jobID {
			state := b.events[i].State
			if state == "SUCCESS" || state == "FAILED" || state == "ABORTED" {
				return &b.events[i]
			}
		}
	}
	return nil
}

// JobEvent represents a job progress event from TrueNAS.
type JobEvent struct {
	ID     int64           `json:"id"`
	State  string          `json:"state"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// WebSocketClient implements Client using channels instead of mutexes.
type WebSocketClient struct {
	config WebSocketConfig
	dialer *websocket.Dialer

	// Channels - the only coordination mechanism
	requestChan    chan wsRequest
	readChan       chan JSONRPCResponse
	eventChan      chan JSONRPCResponse
	disconnectChan chan error
	subscribeChan  chan wsSubscription
	stopChan       chan struct{}
	pongChan       chan struct{} // Receives pong notifications from reader

	testInsecure bool   // For testing with httptest servers
	wsPath       string // Cached WebSocket path

	// Version (cached from fallback during Connect)
	version   api.Version
	connected bool
}

// NewWebSocketClient creates a new channel-based WebSocket client.
func NewWebSocketClient(config WebSocketConfig) (*WebSocketClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: config.ConnectTimeout,
	}
	if config.InsecureSkipVerify {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	c := &WebSocketClient{
		config:         config,
		dialer:         dialer,
		requestChan:    make(chan wsRequest, 100),
		readChan:       make(chan JSONRPCResponse, 100),
		eventChan:      make(chan JSONRPCResponse, 100),
		disconnectChan: make(chan error, 1),
		subscribeChan:  make(chan wsSubscription, 10),
		stopChan:       make(chan struct{}),
		pongChan:       make(chan struct{}, 1),
	}

	// Start writer goroutine
	go c.writerLoop()

	return c, nil
}

// Close stops the client and cleans up resources.
func (c *WebSocketClient) Close() error {
	close(c.stopChan)
	return nil
}

// writerLoop is the main event loop that owns connection state.
func (c *WebSocketClient) writerLoop() {
	var conn *websocket.Conn
	pending := make(map[string]wsRequest)
	jobSubs := make(map[int64]chan<- JobEvent)
	eventBuffer := &jobEventBuffer{}
	var nextID int64

	// Ping/pong state
	var pingTicker *time.Ticker
	var pingTickerChan <-chan time.Time
	var awaitingPong bool
	var pongDeadline time.Time

	if c.config.PingInterval > 0 {
		pingTicker = time.NewTicker(c.config.PingInterval)
		pingTickerChan = pingTicker.C
		defer pingTicker.Stop()
	}

	for {
		select {
		case req := <-c.requestChan:
			// Ensure connected
			if conn == nil {
				var err error
				conn, err = c.connect(req.ctx)
				if err != nil {
					req.response <- wsResponse{err: err}
					continue
				}
				go c.readerLoop(conn)
			}

			// Build JSON-RPC request
			id := fmt.Sprintf("req-%d", nextID)
			nextID++

			rpcReq := JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  req.method,
				Params:  c.wrapParams(req.params),
				ID:      id,
			}

			if err := conn.WriteJSON(rpcReq); err != nil {
				req.response <- wsResponse{err: err}
				conn.Close()
				conn = nil
				awaitingPong = false
				continue
			}

			pending[id] = req

		case sub := <-c.subscribeChan:
			if sub.ch == nil {
				// Unsubscribe
				delete(jobSubs, sub.jobID)
			} else {
				// Subscribe - first check buffer for already-received events
				jobSubs[sub.jobID] = sub.ch
				if buffered := eventBuffer.getByJobID(sub.jobID); buffered != nil {
					// Replay the terminal event we already received
					sub.ch <- *buffered
					delete(jobSubs, sub.jobID)
				}
			}

		case msg := <-c.readChan:
			if req, ok := pending[msg.ID]; ok {
				delete(pending, msg.ID)
				if msg.Error != nil {
					// Check for auth error - triggers reconnect
					if isAuthenticationError(msg.Error) && conn != nil {
						conn.Close()
						conn = nil
						awaitingPong = false
					}
					req.response <- wsResponse{err: msg.Error}
				} else {
					req.response <- wsResponse{result: msg.Result}
				}
			}

		case msg := <-c.eventChan:
			// Handle job events - parse the raw message and buffer for replay
			c.handleJobEvent(msg, jobSubs, eventBuffer)

		case err := <-c.disconnectChan:
			if conn != nil {
				conn.Close()
				conn = nil
			}
			awaitingPong = false
			// Fail all pending requests
			for id, req := range pending {
				req.response <- wsResponse{err: err}
				delete(pending, id)
			}

		case <-c.stopChan:
			if conn != nil {
				conn.Close()
			}
			// Fail remaining pending requests
			for id, req := range pending {
				req.response <- wsResponse{err: errors.New("client closed")}
				delete(pending, id)
			}
			return

		case <-pingTickerChan:
			if conn != nil && !awaitingPong {
				err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(c.config.PingTimeout))
				if err != nil {
					conn.Close()
					conn = nil
					for id, req := range pending {
						req.response <- wsResponse{err: fmt.Errorf("ping failed: %w", err)}
						delete(pending, id)
					}
					continue
				}
				awaitingPong = true
				pongDeadline = time.Now().Add(c.config.PingTimeout)
			}

		case <-c.pongChan:
			awaitingPong = false
		}

		// Check for pong timeout
		if awaitingPong && time.Now().After(pongDeadline) {
			if conn != nil {
				conn.Close()
				conn = nil
			}
			for id, req := range pending {
				req.response <- wsResponse{err: errors.New("pong timeout")}
				delete(pending, id)
			}
			awaitingPong = false
		}
	}
}

// handleJobEvent parses, buffers, and routes job events to subscribers.
func (c *WebSocketClient) handleJobEvent(msg JSONRPCResponse, jobSubs map[int64]chan<- JobEvent, buffer *jobEventBuffer) {
	// TrueNAS sends job events in a nested structure:
	// {
	//   "msg": "method",
	//   "method": "collection_update",
	//   "params": {
	//     "msg": "changed",        // or "added", "removed"
	//     "collection": "core.get_jobs",
	//     "id": 123,
	//     "fields": {"state": "SUCCESS", "result": {...}, "error": "..."}
	//   }
	// }
	// The raw message is stored in msg.Result

	if msg.Result == nil {
		return
	}

	var envelope struct {
		Msg    string `json:"msg"`
		Method string `json:"method"`
		Params struct {
			Msg        string `json:"msg"`
			Collection string `json:"collection"`
			ID         int64  `json:"id"`
			Fields     struct {
				State  string          `json:"state"`
				Result json.RawMessage `json:"result"`
				Error  string          `json:"error"`
			} `json:"fields"`
		} `json:"params"`
	}

	if err := json.Unmarshal(msg.Result, &envelope); err != nil {
		return
	}

	if envelope.Method == "collection_update" && envelope.Params.Collection == "core.get_jobs" {
		event := JobEvent{
			ID:     envelope.Params.ID,
			State:  envelope.Params.Fields.State,
			Result: envelope.Params.Fields.Result,
			Error:  envelope.Params.Fields.Error,
		}

		// Buffer terminal events so new subscribers can find already-completed jobs
		if event.State == "SUCCESS" || event.State == "FAILED" || event.State == "ABORTED" {
			buffer.add(event)
		}

		c.routeJobEvent(event.ID, event.State, event.Result, event.Error, jobSubs)
	}
}

// routeJobEvent sends the event to the appropriate subscriber.
func (c *WebSocketClient) routeJobEvent(jobID int64, state string, result json.RawMessage, errStr string, jobSubs map[int64]chan<- JobEvent) {
	if ch, ok := jobSubs[jobID]; ok {
		ch <- JobEvent{
			ID:     jobID,
			State:  state,
			Result: result,
			Error:  errStr,
		}
		if state == "SUCCESS" || state == "FAILED" || state == "ABORTED" {
			delete(jobSubs, jobID)
		}
	}
}

// wrapParams wraps params for JSON-RPC (single values become arrays).
func (c *WebSocketClient) wrapParams(params any) any {
	if params == nil {
		return nil
	}
	if _, ok := params.([]any); ok {
		return params
	}
	return []any{params}
}

// isAuthenticationError checks if the error indicates session expiry.
func isAuthenticationError(err *JSONRPCError) bool {
	if err.Data != nil {
		return strings.Contains(err.Data.Reason, "ENOTAUTHENTICATED")
	}
	return false
}

// ErrUnsupportedVersion is returned when WebSocket transport is used with TrueNAS < 25.0.
var ErrUnsupportedVersion = errors.New("WebSocket transport requires TrueNAS 25.0 or later. " +
	"TrueNAS 24.x uses a legacy protocol that is not supported. Use auth_method = \"ssh\" instead")

// connect establishes WebSocket connection and authenticates.
func (c *WebSocketClient) connect(ctx context.Context) (*websocket.Conn, error) {
	// Determine path if not cached (use version from Connect)
	if c.wsPath == "" {
		version := c.version
		// TrueNAS 25.0+ uses /api/current with JSON-RPC 2.0 protocol
		// TrueNAS 24.x uses /websocket with a legacy DDP-like protocol (not supported)
		if !version.AtLeast(25, 0) {
			return nil, fmt.Errorf("%w (detected version: %s)", ErrUnsupportedVersion, version.Raw)
		}
		c.wsPath = "/api/current"
	}

	conn, _, err := c.dialer.DialContext(ctx, c.endpoint(), http.Header{})
	if err != nil {
		return nil, fmt.Errorf("websocket connect failed: %w", err)
	}

	// Authenticate
	if err := c.authenticate(ctx, conn); err != nil {
		conn.Close()
		return nil, err
	}

	// Subscribe to job events at connection level.
	// TrueNAS subscriptions are per-collection, not per-job, so we subscribe once
	// and filter locally by job ID. This ensures events are flowing before any
	// job is created, avoiding race conditions.
	if err := c.subscribeJobEvents(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// subscribeJobEvents subscribes to core.get_jobs events on the connection.
func (c *WebSocketClient) subscribeJobEvents(conn *websocket.Conn) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "core.subscribe",
		Params:  []any{"core.get_jobs"},
		ID:      "job-sub",
	}

	if err := conn.WriteJSON(req); err != nil {
		return fmt.Errorf("job subscription write failed: %w", err)
	}

	var resp JSONRPCResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("job subscription read failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("job subscription failed: %s", resp.Error.Error())
	}

	return nil
}

// endpoint returns the WebSocket URL.
func (c *WebSocketClient) endpoint() string {
	scheme := "wss"
	if c.testInsecure {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, c.config.Host, c.config.Port, c.wsPath)
}

// authenticate sends auth.login_ex and waits for response.
func (c *WebSocketClient) authenticate(ctx context.Context, conn *websocket.Conn) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "auth.login_ex",
		Params: []any{map[string]string{
			"mechanism": "API_KEY_PLAIN",
			"username":  c.config.Username,
			"api_key":   c.config.APIKey,
		}},
		ID: "auth",
	}

	if err := conn.WriteJSON(req); err != nil {
		return fmt.Errorf("auth write failed: %w", err)
	}

	var resp JSONRPCResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("auth read failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("authentication failed: %s", resp.Error.Error())
	}

	var result struct {
		ResponseType string `json:"response_type"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("auth response parse failed: %w", err)
	}
	if result.ResponseType != "SUCCESS" {
		return fmt.Errorf("authentication failed: %s", result.ResponseType)
	}

	return nil
}

// readerLoop reads messages and forwards to writer.
func (c *WebSocketClient) readerLoop(conn *websocket.Conn) {
	conn.SetPongHandler(func(appData string) error {
		select {
		case c.pongChan <- struct{}{}:
		default:
		}
		return nil
	})

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			select {
			case c.disconnectChan <- err:
			default:
			}
			return
		}

		// Try to parse as JSON-RPC response first
		var rpcResp JSONRPCResponse
		if err := json.Unmarshal(rawMsg, &rpcResp); err == nil && rpcResp.ID != "" {
			c.readChan <- rpcResp
			continue
		}

		// Otherwise, treat as an event (push message)
		// Store raw bytes in Result for later parsing
		eventMsg := JSONRPCResponse{
			Result: rawMsg,
		}
		c.eventChan <- eventMsg
	}
}

// Call executes a method with retry logic.
func (c *WebSocketClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	var lastErr error
	classifier := &WebSocketRetryClassifier{}

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := CalculateBackoff(attempt)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := c.doCall(ctx, method, params)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !classifier.IsRetriable(err) {
			return nil, err
		}
	}

	return nil, lastErr
}

// doCall performs a single call attempt.
func (c *WebSocketClient) doCall(ctx context.Context, method string, params any) (json.RawMessage, error) {
	respChan := make(chan wsResponse, 1)

	req := wsRequest{
		method:   method,
		params:   params,
		response: respChan,
		ctx:      ctx,
	}

	select {
	case c.requestChan <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case resp := <-respChan:
		return resp.result, resp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CallAndWait executes a method and waits for job completion.
func (c *WebSocketClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	result, err := c.Call(ctx, method, params)
	if err != nil {
		return nil, err
	}

	// Check if result is a job ID
	var jobID int64
	if err := json.Unmarshal(result, &jobID); err != nil {
		return result, nil // Not a job ID, return directly
	}

	// Subscribe to job events locally.
	// Note: The collection subscription (core.subscribe("core.get_jobs")) happens
	// at connection time, so events are already flowing. This just registers our
	// interest in a specific job ID.
	//
	// RACE CONDITION HANDLING: If the job completed before we subscribed,
	// writerLoop will replay the buffered terminal event to our channel
	// immediately when we register. This eliminates the race window between
	// the method returning and the subscription being registered.
	eventChan := make(chan JobEvent, 10)
	c.subscribeJob(ctx, jobID, eventChan)
	defer c.unsubscribeJob(jobID)

	// Wait for completion via events
	for {
		select {
		case event := <-eventChan:
			switch event.State {
			case "SUCCESS":
				return event.Result, nil
			case "FAILED", "ABORTED":
				if event.Error != "" {
					tnErr := ParseTrueNASError(event.Error)
					// Enrich with app lifecycle log if applicable (uses fallback SSH client with sudo)
					EnrichAppLifecycleError(ctx, tnErr, func(ctx context.Context, path string) (string, error) {
						content, err := c.ReadFile(ctx, path)
						return string(content), err
					})
					return nil, tnErr
				}
				return nil, fmt.Errorf("job %d failed", jobID)
			}
			// RUNNING, WAITING - continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// subscribeJob registers for job events locally.
// Note: The actual core.subscribe("core.get_jobs") happens at connection time.
// This function only registers the local channel to receive filtered events.
func (c *WebSocketClient) subscribeJob(ctx context.Context, jobID int64, ch chan<- JobEvent) {
	sub := wsSubscription{jobID: jobID, ch: ch}
	select {
	case c.subscribeChan <- sub:
	case <-ctx.Done():
		return
	}
}

// unsubscribeJob removes job event registration.
func (c *WebSocketClient) unsubscribeJob(jobID int64) {
	// Send unsubscribe message via subscribeChan with nil channel
	sub := wsSubscription{jobID: jobID, ch: nil}
	select {
	case c.subscribeChan <- sub:
	default:
	}
}

// Connect establishes connection via fallback client and caches version.
// Must be called before using the client.
func (c *WebSocketClient) Connect(ctx context.Context) error {
	if err := c.config.Fallback.Connect(ctx); err != nil {
		return err
	}
	c.version = c.config.Fallback.Version()
	c.connected = true
	return nil
}

// Version returns the cached TrueNAS version.
// Panics if called before Connect() - fail fast on programmer error.
func (c *WebSocketClient) Version() api.Version {
	if !c.connected {
		panic("client.Version() called before Connect()")
	}
	return c.version
}

// WriteFile writes content to a file using filesystem.file_receive.
func (c *WebSocketClient) WriteFile(ctx context.Context, path string, params WriteFileParams) error {
	b64Content := base64.StdEncoding.EncodeToString(params.Content)

	uid := -1
	if params.UID != nil {
		uid = *params.UID
	}
	gid := -1
	if params.GID != nil {
		gid = *params.GID
	}

	apiParams := []any{
		path,
		b64Content,
		map[string]any{
			"mode": int(params.Mode),
			"uid":  uid,
			"gid":  gid,
		},
	}

	_, err := c.Call(ctx, "filesystem.file_receive", apiParams)
	if err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}
	return nil
}

// ReadFile delegates to fallback SSH client.
func (c *WebSocketClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return c.config.Fallback.ReadFile(ctx, path)
}

// DeleteFile delegates to fallback SSH client.
func (c *WebSocketClient) DeleteFile(ctx context.Context, path string) error {
	return c.config.Fallback.DeleteFile(ctx, path)
}

// RemoveDir delegates to fallback SSH client.
func (c *WebSocketClient) RemoveDir(ctx context.Context, path string) error {
	return c.config.Fallback.RemoveDir(ctx, path)
}

// RemoveAll delegates to fallback SSH client.
func (c *WebSocketClient) RemoveAll(ctx context.Context, path string) error {
	return c.config.Fallback.RemoveAll(ctx, path)
}

// FileExists checks if a file exists using filesystem.stat.
func (c *WebSocketClient) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := c.Call(ctx, "filesystem.stat", path)
	if err != nil {
		var rpcErr *JSONRPCError
		if errors.As(err, &rpcErr) && rpcErr.Data != nil && rpcErr.Data.Error == 2 {
			return false, nil // ENOENT
		}
		return false, fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	return true, nil
}

// Chown changes ownership using filesystem.chown.
func (c *WebSocketClient) Chown(ctx context.Context, path string, uid, gid int) error {
	params := map[string]any{
		"path": path,
		"uid":  uid,
		"gid":  gid,
	}
	_, err := c.CallAndWait(ctx, "filesystem.chown", params)
	if err != nil {
		return fmt.Errorf("failed to chown %q: %w", path, err)
	}
	return nil
}

// ChmodRecursive changes permissions using filesystem.setperm.
func (c *WebSocketClient) ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error {
	params := map[string]any{
		"path": path,
		"mode": fmt.Sprintf("%04o", mode),
		"options": map[string]any{
			"recursive": true,
		},
	}
	_, err := c.CallAndWait(ctx, "filesystem.setperm", params)
	if err != nil {
		return fmt.Errorf("failed to chmod %q: %w", path, err)
	}
	return nil
}

// MkdirAll creates a directory using filesystem.mkdir.
func (c *WebSocketClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	params := map[string]any{
		"path": path,
		"options": map[string]any{
			"mode": fmt.Sprintf("%04o", mode),
		},
	}
	_, err := c.Call(ctx, "filesystem.mkdir", params)
	if err != nil {
		return fmt.Errorf("failed to mkdir %q: %w", path, err)
	}
	return nil
}
