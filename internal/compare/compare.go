package compare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/Alarm2024/stale/internal/redact"
)

type Endpoint struct {
	Label string
	URL   string
}

type Options struct {
	A      Endpoint
	B      Endpoint
	Every  time.Duration
	Rounds int
	Out    io.Writer
	Now    func() time.Time
	Sleep  func(context.Context, time.Duration) error
	Call   func(ctx context.Context, endpoint, method string) CallResult
}

type roundStats struct {
	healthA, healthB, slotA, slotB CallResult
}

func Run(ctx context.Context, opts Options) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Every <= 0 {
		return errors.New("interval must be positive")
	}
	if opts.Rounds <= 0 {
		return errors.New("rounds must be positive")
	}
	if opts.A.URL == "" || opts.B.URL == "" {
		return errors.New("STALE_A and STALE_B must be set")
	}
	opts.A.Label = normalizeLabel(opts.A.Label)
	opts.B.Label = normalizeLabel(opts.B.Label)
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Sleep == nil {
		opts.Sleep = sleepContext
	}
	caller := newRPCCaller(defaultCallTimeout)
	if opts.Call == nil {
		opts.Call = caller.Call
	}

	fpA := redact.Fingerprint(opts.A.URL)
	fpB := redact.Fingerprint(opts.B.URL)

	fmt.Fprintf(opts.Out, "stale compare rounds=%d every=%s %s=%s %s=%s\n",
		opts.Rounds, opts.Every.Round(time.Millisecond),
		opts.A.Label, fpA, opts.B.Label, fpB)

	repA := &endpointReport{label: opts.A.Label, fp: fpA, errors: map[string]int{}, rounds: opts.Rounds}
	repB := &endpointReport{label: opts.B.Label, fp: fpB, errors: map[string]int{}, rounds: opts.Rounds}

	for round := 1; round <= opts.Rounds; round++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		stats := probeRound(ctx, opts)
		printRound(opts.Out, round, opts.A, fpA, opts.B, fpB, stats)

		accumulateErrors(repA, stats.healthA)
		accumulateErrors(repA, stats.slotA)
		accumulateErrors(repB, stats.healthB)
		accumulateErrors(repB, stats.slotB)

		if stats.slotA.ErrorClass == "" {
			repA.addSlotOK(stats.slotA.Latency)
		}
		if stats.slotB.ErrorClass == "" {
			repB.addSlotOK(stats.slotB.Latency)
		}

		lagA, lagKnownA := slotLagBehindPeer(stats.slotA.Slot, stats.slotB.Slot, stats.slotA.SlotOK, stats.slotB.SlotOK)
		lagB, lagKnownB := slotLagBehindPeer(stats.slotB.Slot, stats.slotA.Slot, stats.slotB.SlotOK, stats.slotA.SlotOK)
		if lagKnownA {
			repA.addLag(lagA)
		}
		if lagKnownB {
			repB.addLag(lagB)
		}

		if auth := authStopClass(stats); auth != "" {
			printEndpointReports(opts.Out, repA, repB)
			return fmt.Errorf("probe stopped after HTTP %s (no further rounds)", auth)
		}

		if round < opts.Rounds {
			if err := opts.Sleep(ctx, opts.Every); err != nil {
				return err
			}
		}
	}

	printEndpointReports(opts.Out, repA, repB)
	return nil
}

func authStopClass(s roundStats) string {
	for _, r := range []CallResult{s.healthA, s.slotA, s.healthB, s.slotB} {
		if r.ErrorClass == "401" || r.ErrorClass == "403" {
			return r.ErrorClass
		}
	}
	return ""
}

func probeRound(ctx context.Context, opts Options) roundStats {
	type job struct {
		k    string
		ep   Endpoint
		meth string
	}
	jobs := []job{
		{"healthA", opts.A, "getHealth"},
		{"slotA", opts.A, "getSlot"},
		{"healthB", opts.B, "getHealth"},
		{"slotB", opts.B, "getSlot"},
	}
	type result struct {
		k string
		r CallResult
	}
	ch := make(chan result, len(jobs))
	for _, job := range jobs {
		job := job
		go func() {
			ch <- result{k: job.k, r: opts.Call(ctx, job.ep.URL, job.meth)}
		}()
	}
	var stats roundStats
	for range jobs {
		res := <-ch
		switch res.k {
		case "healthA":
			stats.healthA = res.r
		case "slotA":
			stats.slotA = res.r
		case "healthB":
			stats.healthB = res.r
		case "slotB":
			stats.slotB = res.r
		}
	}
	return stats
}

func printRound(w io.Writer, round int, a Endpoint, fpA string, b Endpoint, fpB string, stats roundStats) {
	lagA, lagKnownA := slotLagBehindPeer(stats.slotA.Slot, stats.slotB.Slot, stats.slotA.SlotOK, stats.slotB.SlotOK)
	lagB, lagKnownB := slotLagBehindPeer(stats.slotB.Slot, stats.slotA.Slot, stats.slotB.SlotOK, stats.slotA.SlotOK)

	printProbe(w, round, a.Label, fpA, "getHealth", stats.healthA, 0, false)
	printProbe(w, round, a.Label, fpA, "getSlot", stats.slotA, lagA, lagKnownA)
	printProbe(w, round, b.Label, fpB, "getHealth", stats.healthB, 0, false)
	printProbe(w, round, b.Label, fpB, "getSlot", stats.slotB, lagB, lagKnownB)
}

func printProbe(w io.Writer, round int, label, fp, method string, res CallResult, lag int64, lagOK bool) {
	fmt.Fprintf(w, "round=%d label=%s fp=%s method=%s status=%s latency=%s slot=%s lag=%s err=%s\n",
		round, label, fp, method,
		formatStatus(res.HTTPStatus),
		formatLatency(res.Latency),
		formatSlot(res.Slot, res.SlotOK),
		formatLag(lag, lagOK),
		formatErrClass(res.ErrorClass),
	)
}

func accumulateErrors(rep *endpointReport, res CallResult) {
	rep.bumpErr(res.ErrorClass)
}

func printEndpointReports(w io.Writer, reps ...*endpointReport) {
	fmt.Fprintln(w, "definitions latency_p50_p90=nearest-rank on successful getSlot samples; lag_slots=nearest-rank on max(0,peer-self) per round when both slots OK")
	sort.Slice(reps, func(i, j int) bool { return reps[i].label < reps[j].label })
	for _, rep := range reps {
		printOneEndpoint(w, rep)
	}
}

func printOneEndpoint(w io.Writer, rep *endpointReport) {
	n := len(rep.slotSamples)
	p50, p50OK := percentileRank(rep.slotSamples, 0.50)
	p90, p90OK := percentileRank(rep.slotSamples, 0.90)
	lagP50, lagP90, lagMax, lagOK := lagAggregate(rep.lagSamples)

	latP50 := "-"
	latP90 := "-"
	if p50OK {
		latP50 = formatLatency(p50)
	}
	if p90OK {
		latP90 = formatLatency(p90)
	}
	lagStr := "-"
	if lagOK {
		lagStr = fmt.Sprintf("p50=%s p90=%s max=%s", formatInt64(lagP50), formatInt64(lagP90), formatInt64(lagMax))
	}

	fmt.Fprintf(w, "endpoint label=%s fp=%s getSlot_N=%d latency_p50=%s latency_p90=%s lag_vs_peer_slots=%s",
		rep.label, rep.fp, n, latP50, latP90, lagStr)
	errParts := formatErrorCounts(rep.errors)
	if errParts != "" {
		fmt.Fprintf(w, " errors=%s", errParts)
	}
	fmt.Fprintln(w)
}

func formatErrorCounts(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	order := []string{"timeout", "429", "5xx", "401", "403"}
	parts := make([]string, 0, len(m))
	seen := map[string]bool{}
	for _, cls := range order {
		if n, ok := m[cls]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", cls, n))
			seen[cls] = true
		}
	}
	rest := make([]string, 0)
	for cls, n := range m {
		if seen[cls] || n == 0 {
			continue
		}
		rest = append(rest, fmt.Sprintf("%s:%d", cls, n))
	}
	sort.Strings(rest)
	parts = append(parts, rest...)
	return joinComma(parts)
}

func joinComma(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "," + p
	}
	return out
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
