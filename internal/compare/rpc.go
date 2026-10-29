package compare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultCallTimeout = 2 * time.Second

type CallResult struct {
	HTTPStatus int
	Latency    time.Duration
	Slot       uint64
	SlotOK     bool
	ErrorClass string
}

// ClassifyError maps an HTTP status and/or error into the compare error class
// strings printed in output.
func ClassifyError(httpStatus int, err error, rpcCode int) string {
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "timeout"
		}
		if rpcCode != 0 {
			return formatRPCCode(rpcCode)
		}
		return "network"
	}
	if rpcCode != 0 {
		return formatRPCCode(rpcCode)
	}
	if httpStatus == 0 {
		return "network"
	}
	if cls := ClassifyHTTPStatus(httpStatus); cls != "" {
		return cls
	}
	return ""
}

func ClassifyHTTPStatus(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "401"
	case http.StatusForbidden:
		return "403"
	case http.StatusTooManyRequests:
		return "429"
	default:
		if status >= 500 {
			return "5xx"
		}
	}
	if status >= 400 {
		return fmt.Sprintf("http-%d", status)
	}
	return ""
}

func formatRPCCode(code int) string {
	return fmt.Sprintf("rpc %d", code)
}

type RPCCaller struct {
	client  *http.Client
	timeout time.Duration
}

func NewRPCCaller(timeout time.Duration) *RPCCaller {
	if timeout <= 0 {
		timeout = defaultCallTimeout
	}
	return &RPCCaller{
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

func newRPCCaller(timeout time.Duration) *RPCCaller {
	return NewRPCCaller(timeout)
}

func (c *RPCCaller) Call(ctx context.Context, endpoint, method string) CallResult {
	start := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	body, err := json.Marshal(rpcRequest(method))
	if err != nil {
		return CallResult{Latency: time.Since(start), ErrorClass: "network"}
	}

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return CallResult{Latency: time.Since(start), ErrorClass: "network"}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		cls := ClassifyError(0, callCtx.Err(), 0)
		if cls == "" {
			cls = "network"
		}
		return CallResult{Latency: latency, ErrorClass: cls}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	status := resp.StatusCode

	if status < 200 || status >= 300 {
		cls := ClassifyHTTPStatus(status)
		if cls == "" {
			cls = fmt.Sprintf("http-%d", status)
		}
		return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: cls}
	}

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: "network"}
	}
	if envelope.Error != nil {
		cls := formatRPCCode(envelope.Error.Code)
		return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: cls}
	}

	switch method {
	case "getHealth":
		var health string
		if err := json.Unmarshal(envelope.Result, &health); err != nil || health != "ok" {
			return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: formatRPCCode(-32000)}
		}
		return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: ""}
	case "getSlot":
		var slot uint64
		if err := json.Unmarshal(envelope.Result, &slot); err != nil {
			return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: "network"}
		}
		return CallResult{HTTPStatus: status, Latency: latency, Slot: slot, SlotOK: true, ErrorClass: ""}
	default:
		return CallResult{HTTPStatus: status, Latency: latency, ErrorClass: "network"}
	}
}

func rpcRequest(method string) map[string]any {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if method == "getSlot" {
		req["params"] = []any{map[string]any{"commitment": "processed"}}
	}
	return req
}

func slotLag(self, peer uint64, selfOK, peerOK bool) (lag int64, known bool) {
	if !selfOK || !peerOK {
		return 0, false
	}
	if peer < self {
		return 0, true
	}
	return int64(peer - self), true
}

func formatErrClass(cls string) string {
	if cls == "" {
		return "-"
	}
	return cls
}

func formatSlot(slot uint64, ok bool) string {
	if !ok {
		return "-"
	}
	return strconv.FormatUint(slot, 10)
}

func formatLag(lag int64, known bool) string {
	if !known {
		return "-"
	}
	return strconv.FormatInt(lag, 10)
}

func formatStatus(status int) string {
	if status == 0 {
		return "-"
	}
	return strconv.Itoa(status)
}

func formatLatency(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1 && d > 0 {
		return "<1ms"
	}
	return fmt.Sprintf("%dms", ms)
}

func normalizeLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "?"
	}
	return s
}
