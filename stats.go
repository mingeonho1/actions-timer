package main

import (
	"sort"
	"time"
)

// Stats summarizes durations for one workflow.
// 워크플로 하나의 소요시간 요약.
type Stats struct {
	Workflow    string        `json:"workflow"`
	Runs        int           `json:"runs"`
	SuccessRate float64       `json:"success_rate"`
	Avg         time.Duration `json:"avg_ns"`
	P50         time.Duration `json:"p50_ns"`
	P95         time.Duration `json:"p95_ns"`
	Max         time.Duration `json:"max_ns"`
}

// Summarize groups runs by workflow name, slowest p95 first.
// 워크플로 이름별로 묶고, p95가 느린 순으로 정렬한다.
func Summarize(runs []Run) []Stats {
	groups := map[string][]Run{}
	for _, r := range runs {
		groups[r.Name] = append(groups[r.Name], r)
	}

	out := make([]Stats, 0, len(groups))
	for name, rs := range groups {
		ds := make([]time.Duration, len(rs))
		var total time.Duration
		success := 0
		for i, r := range rs {
			ds[i] = r.Duration()
			total += ds[i]
			if r.Conclusion == "success" {
				success++
			}
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		out = append(out, Stats{
			Workflow:    name,
			Runs:        len(rs),
			SuccessRate: float64(success) / float64(len(rs)),
			Avg:         total / time.Duration(len(rs)),
			P50:         percentile(ds, 50),
			P95:         percentile(ds, 95),
			Max:         ds[len(ds)-1],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].P95 != out[j].P95 {
			return out[i].P95 > out[j].P95
		}
		return out[i].Workflow < out[j].Workflow
	})
	return out
}

// percentile uses nearest-rank on an ascending slice.
// 정렬된 값에서 nearest-rank 방식으로 백분위를 고른다.
func percentile(sorted []time.Duration, p int) time.Duration {
	rank := (p*len(sorted) + 99) / 100 // ceil(p/100 * n)
	if rank < 1 {
		rank = 1
	}
	return sorted[rank-1]
}
