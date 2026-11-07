package compare

import (
	"fmt"
	"sort"
	"time"
)

// percentileRank returns latency at the nearest-rank percentile: sort ascending,
// pick the element at index ceil(p*n)-1 (1-based rank), clamped to [0,n-1].
// p50/p90 in summaries use p=0.50 and p=0.90 on successful getSlot samples only.
func percentileRank(latencies []time.Duration, p float64) (time.Duration, bool) {
	n := len(latencies)
	if n == 0 {
		return 0, false
	}
	cp := append([]time.Duration(nil), latencies...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(n)*p + 0.999999999) // ceil(p*n) - 1 via ceil
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return cp[idx], true
}

// slotLagBehindPeer is slots this endpoint trails the other when both getSlot
// calls succeed in the same round: max(0, peer_slot - self_slot). When either
// slot is missing, that round contributes no lag sample.
func slotLagBehindPeer(self, peer uint64, selfOK, peerOK bool) (int64, bool) {
	if !selfOK || !peerOK {
		return 0, false
	}
	if peer < self {
		return 0, true
	}
	return int64(peer - self), true
}

type endpointReport struct {
	label       string
	fp          string
	slotSamples []time.Duration
	lagSamples  []int64
	errors      map[string]int
	rounds      int
}

func (r *endpointReport) addSlotOK(lat time.Duration) {
	r.slotSamples = append(r.slotSamples, lat)
}

func (r *endpointReport) addLag(lag int64) {
	r.lagSamples = append(r.lagSamples, lag)
}

func (r *endpointReport) bumpErr(cls string) {
	if cls == "" {
		return
	}
	if r.errors == nil {
		r.errors = map[string]int{}
	}
	r.errors[cls]++
}

func formatInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}

func lagAggregate(lags []int64) (p50 int64, p90 int64, max int64, ok bool) {
	if len(lags) == 0 {
		return 0, 0, 0, false
	}
	cp := append([]int64(nil), lags...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	n := len(cp)
	p50 = cp[percentileIndex(n, 0.50)]
	p90 = cp[percentileIndex(n, 0.90)]
	max = cp[n-1]
	return p50, p90, max, true
}

func percentileIndex(n int, p float64) int {
	idx := int(float64(n)*p+0.999999999) - 1
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}
