package compare_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Alarm2024/stale/internal/compare"
)

func TestFastVersusSlowLatencyPercentiles(t *testing.T) {
	aURL := "http://fast"
	bURL := "http://slow"

	var out strings.Builder
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "fast", URL: aURL},
		B:      compare.Endpoint{Label: "slow", URL: bURL},
		Rounds: 4,
		Every:  time.Millisecond,
		Out:    &out,
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			if method != "getSlot" {
				return compare.CallResult{HTTPStatus: 200}
			}
			switch endpoint {
			case aURL:
				return compare.CallResult{HTTPStatus: 200, Latency: 10 * time.Millisecond, Slot: 100, SlotOK: true}
			case bURL:
				return compare.CallResult{HTTPStatus: 200, Latency: 100 * time.Millisecond, Slot: 100, SlotOK: true}
			default:
				return compare.CallResult{ErrorClass: "network"}
			}
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "label=fast") || !strings.Contains(text, "latency_p50=10ms") {
		t.Fatalf("fast endpoint p50:\n%s", text)
	}
	if !strings.Contains(text, "label=slow") || !strings.Contains(text, "latency_p90=100ms") {
		t.Fatalf("slow endpoint p90:\n%s", text)
	}
}

func TestLaggingSlotVersusPeer(t *testing.T) {
	aURL := "http://a"
	bURL := "http://b"

	var out strings.Builder
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: aURL},
		B:      compare.Endpoint{Label: "B", URL: bURL},
		Rounds: 2,
		Every:  time.Millisecond,
		Out:    &out,
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			if method != "getSlot" {
				return compare.CallResult{HTTPStatus: 200}
			}
			if endpoint == aURL {
				return compare.CallResult{HTTPStatus: 200, Latency: time.Millisecond, Slot: 1000, SlotOK: true}
			}
			return compare.CallResult{HTTPStatus: 200, Latency: time.Millisecond, Slot: 1015, SlotOK: true}
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "label=A") || !strings.Contains(text, "lag_vs_peer_slots=p50=15") {
		t.Fatalf("expected A lag p50=15:\n%s", text)
	}
}

func Test429StormCounted(t *testing.T) {
	aURL := "http://a"
	bURL := "http://b"
	n429 := 0

	var out strings.Builder
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: aURL},
		B:      compare.Endpoint{Label: "B", URL: bURL},
		Rounds: 3,
		Every:  time.Millisecond,
		Out:    &out,
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			if endpoint == bURL {
				return compare.CallResult{HTTPStatus: 200, Latency: time.Millisecond, Slot: 1, SlotOK: true}
			}
			if method == "getSlot" {
				n429++
				return compare.CallResult{HTTPStatus: 429, Latency: time.Millisecond, ErrorClass: "429"}
			}
			return compare.CallResult{HTTPStatus: 200}
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "429:3") || !strings.Contains(text, "getSlot_N=0") || !strings.Contains(text, "latency_p50=-") {
		t.Fatalf("429 storm summary:\n%s", text)
	}
}

func Test401StopsProbing(t *testing.T) {
	aURL := "http://a"
	bURL := "http://b"
	rounds := 0

	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: aURL},
		B:      compare.Endpoint{Label: "B", URL: bURL},
		Rounds: 5,
		Every:  time.Millisecond,
		Out:    ioDiscard{},
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			rounds++
			if endpoint == aURL && method == "getSlot" {
				return compare.CallResult{HTTPStatus: 401, ErrorClass: "401"}
			}
			return compare.CallResult{HTTPStatus: 200, Latency: time.Millisecond, Slot: 1, SlotOK: true}
		},
	})
	if err == nil {
		t.Fatal("expected stop error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
	if rounds != 4 { // round 1: 4 parallel calls
		t.Fatalf("rounds executed calls = %d, want 4 (single round then stop)", rounds)
	}
}

func Test403StopsProbing(t *testing.T) {
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: "http://a"},
		B:      compare.Endpoint{Label: "B", URL: "http://b"},
		Rounds: 3,
		Every:  time.Millisecond,
		Out:    ioDiscard{},
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			if method == "getHealth" && endpoint == "http://b" {
				return compare.CallResult{HTTPStatus: 403, ErrorClass: "403"}
			}
			return compare.CallResult{HTTPStatus: 200, Latency: time.Millisecond, Slot: 1, SlotOK: true}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 stop, err=%v", err)
	}
}

func TestInsufficientSuccessfulSamplesShowDash(t *testing.T) {
	var out strings.Builder
	err := compare.Run(context.Background(), compare.Options{
		A:      compare.Endpoint{Label: "A", URL: "http://a"},
		B:      compare.Endpoint{Label: "B", URL: "http://b"},
		Rounds: 1,
		Every:  time.Millisecond,
		Out:    &out,
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Call: func(ctx context.Context, endpoint, method string) compare.CallResult {
			if method == "getSlot" {
				return compare.CallResult{HTTPStatus: 503, ErrorClass: "5xx"}
			}
			return compare.CallResult{HTTPStatus: 200}
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "getSlot_N=0 latency_p50=- latency_p90=- lag_vs_peer_slots=-") {
		t.Fatalf("missing data should not read as zero:\n%s", text)
	}
	if strings.Count(text, "5xx:1") < 2 {
		t.Fatalf("5xx errors per endpoint:\n%s", text)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
