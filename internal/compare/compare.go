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

type summaryKey struct {
	label  string
	method string
}

type summaryLine struct {
	label  string
	method string
	ok     int
	errors map[string]int
	latSum time.Duration
	maxLag int64
	hasLag bool
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

	totals := map[summaryKey]*summaryLine{
		{opts.A.Label, "getHealth"}: {label: opts.A.Label, method: "getHealth", errors: map[string]int{}},
		{opts.A.Label, "getSlot"}:   {label: opts.A.Label, method: "getSlot", errors: map[string]int{}},
		{opts.B.Label, "getHealth"}: {label: opts.B.Label, method: "getHealth", errors: map[string]int{}},
		{opts.B.Label, "getSlot"}:   {label: opts.B.Label, method: "getSlot", errors: map[string]int{}},
	}

	for round := 1; round <= opts.Rounds; round++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		stats := probeRound(ctx, opts)
		printRound(opts.Out, round, opts.A, fpA, opts.B, fpB, stats)

		accumulate(totals, opts.A.Label, "getHealth", stats.healthA)
		accumulate(totals, opts.A.Label, "getSlot", stats.slotA)
		accumulate(totals, opts.B.Label, "getHealth", stats.healthB)
		accumulate(totals, opts.B.Label, "getSlot", stats.slotB)

		lagA, lagKnownA := slotLag(stats.slotA.Slot, stats.slotB.Slot, stats.slotA.SlotOK, stats.slotB.SlotOK)
		lagB, lagKnownB := slotLag(stats.slotB.Slot, stats.slotA.Slot, stats.slotB.SlotOK, stats.slotA.SlotOK)
		bumpMaxLag(totals, opts.A.Label, lagA, lagKnownA)
		bumpMaxLag(totals, opts.B.Label, lagB, lagKnownB)

		if round < opts.Rounds {
			if err := opts.Sleep(ctx, opts.Every); err != nil {
				return err
			}
		}
	}

	printSummary(opts.Out, opts.Rounds, opts.Every, totals)
	return nil
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
	lagA, lagKnownA := slotLag(stats.slotA.Slot, stats.slotB.Slot, stats.slotA.SlotOK, stats.slotB.SlotOK)
	lagB, lagKnownB := slotLag(stats.slotB.Slot, stats.slotA.Slot, stats.slotB.SlotOK, stats.slotA.SlotOK)

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

func accumulate(totals map[summaryKey]*summaryLine, label, method string, res CallResult) {
	line := totals[summaryKey{label, method}]
	if res.ErrorClass == "" {
		line.ok++
		line.latSum += res.Latency
		return
	}
	line.errors[res.ErrorClass]++
}

func bumpMaxLag(totals map[summaryKey]*summaryLine, label string, lag int64, ok bool) {
	if !ok {
		return
	}
	line := totals[summaryKey{label, "getSlot"}]
	line.hasLag = true
	if lag > line.maxLag {
		line.maxLag = lag
	}
}

func printSummary(w io.Writer, rounds int, every time.Duration, totals map[summaryKey]*summaryLine) {
	fmt.Fprintf(w, "summary rounds=%d every=%s\n", rounds, every.Round(time.Millisecond))
	keys := make([]summaryKey, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].label != keys[j].label {
			return keys[i].label < keys[j].label
		}
		return keys[i].method < keys[j].method
	})
	for _, k := range keys {
		line := totals[k]
		avg := "-"
		if line.ok > 0 {
			avg = formatLatency(line.latSum / time.Duration(line.ok))
		}
		maxLag := "-"
		if line.method == "getSlot" && line.hasLag {
			maxLag = strconvFormatInt(line.maxLag)
		}
		fmt.Fprintf(w, "  label=%s method=%s ok=%d/%d avg_latency=%s max_lag=%s",
			line.label, line.method, line.ok, rounds, avg, maxLag)
		if len(line.errors) > 0 {
			parts := make([]string, 0, len(line.errors))
			for cls, n := range line.errors {
				parts = append(parts, fmt.Sprintf("%s:%d", cls, n))
			}
			sort.Strings(parts)
			fmt.Fprintf(w, " errors=%s", joinComma(parts))
		}
		fmt.Fprintln(w)
	}
}

func strconvFormatInt(v int64) string {
	return fmt.Sprintf("%d", v)
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
