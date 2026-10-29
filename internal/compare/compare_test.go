package compare_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Alarm2024/stale/internal/compare"
)

type fakeRPC struct {
	hang       bool
	httpStatus int
	rpcCode    int
	slot       atomic.Uint64
	healthOK   bool
}

func (f *fakeRPC) handler(w http.ResponseWriter, r *http.Request) {
	if f.hang {
		time.Sleep(5 * time.Second)
		return
	}
	if f.httpStatus != 0 {
		http.Error(w, "error", f.httpStatus)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &req)

	if f.rpcCode != 0 {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]any{
				"code":    f.rpcCode,
				"message": "simulated",
			},
		})
		return
	}

	switch req.Method {
	case "getHealth":
		val := "ok"
		if !f.healthOK {
			val = "behind"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": val})
	case "getSlot":
		slot := f.slot.Load()
		if slot == 0 {
			slot = 1000
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": slot})
	default:
		http.Error(w, "bad method", http.StatusBadRequest)
	}
}

func newFakeServer(f *fakeRPC) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(f.handler))
}

func TestClassifyErrorClasses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		rpc    int
		err    error
		want   string
	}{
		{"timeout", 0, 0, context.DeadlineExceeded, "timeout"},
		{"401", http.StatusUnauthorized, 0, nil, "401"},
		{"403", http.StatusForbidden, 0, nil, "403"},
		{"429", http.StatusTooManyRequests, 0, nil, "429"},
		{"5xx", http.StatusBadGateway, 0, nil, "5xx"},
		{"rpc", http.StatusOK, -32003, nil, "rpc -32003"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compare.ClassifyError(tc.status, tc.err, tc.rpc)
			if got != tc.want {
				t.Fatalf("ClassifyError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCallErrorClasses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		fake    fakeRPC
		wantErr string
	}{
		{"timeout", fakeRPC{hang: true}, "timeout"},
		{"401", fakeRPC{httpStatus: http.StatusUnauthorized}, "401"},
		{"403", fakeRPC{httpStatus: http.StatusForbidden}, "403"},
		{"429", fakeRPC{httpStatus: http.StatusTooManyRequests}, "429"},
		{"5xx", fakeRPC{httpStatus: http.StatusServiceUnavailable}, "5xx"},
		{"rpc", fakeRPC{rpcCode: -32005}, "rpc -32005"},
	}
	for i := range cases {
		tc := &cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newFakeServer(&tc.fake)
			defer srv.Close()

			caller := compare.NewRPCCaller(200 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			for _, method := range []string{"getHealth", "getSlot"} {
				res := caller.Call(ctx, srv.URL, method)
				if res.ErrorClass != tc.wantErr {
					t.Fatalf("%s ErrorClass = %q, want %q", method, res.ErrorClass, tc.wantErr)
				}
			}
		})
	}
}

func TestRunTwoRoundsSummary(t *testing.T) {
	a := newFakeServer(&fakeRPC{healthOK: true})
	defer a.Close()
	bSrv := &fakeRPC{healthOK: true}
	bSrv.slot.Store(1005)
	b2 := newFakeServer(bSrv)
	defer b2.Close()

	var out strings.Builder
	virtual := time.Unix(0, 0)
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "primary", URL: a.URL},
		B:      compare.Endpoint{Label: "backup", URL: b2.URL},
		Every:  time.Second,
		Rounds: 2,
		Out:    &out,
		Now:    func() time.Time { return virtual },
		Sleep: func(ctx context.Context, d time.Duration) error {
			virtual = virtual.Add(d)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "method=getHealth") || !strings.Contains(text, "method=getSlot") {
		t.Fatalf("missing probe lines:\n%s", text)
	}
	if !strings.Contains(text, "summary rounds=2") {
		t.Fatalf("missing summary:\n%s", text)
	}
	if strings.Contains(text, "http://") || strings.Contains(text, "127.0.0.1") {
		t.Fatalf("output leaked endpoint URL:\n%s", text)
	}
}

func TestSlotLagPrintedWhenPeerAhead(t *testing.T) {
	aURL := "http://fake-a"
	bURL := "http://fake-b"

	var out strings.Builder
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: aURL},
		B:      compare.Endpoint{Label: "B", URL: bURL},
		Every:  time.Second,
		Rounds: 1,
		Out:    &out,
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			switch method {
			case "getHealth":
				return compare.CallResult{HTTPStatus: 200}
			case "getSlot":
				if endpoint == aURL {
					return compare.CallResult{HTTPStatus: 200, Slot: 1000, SlotOK: true}
				}
				return compare.CallResult{HTTPStatus: 200, Slot: 1010, SlotOK: true}
			default:
				return compare.CallResult{ErrorClass: "network"}
			}
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "label=A") || !strings.Contains(text, "lag=10") {
		t.Fatalf("expected A lag=10 when peer ahead:\n%s", text)
	}
}
