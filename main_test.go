package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func runAt(name, conclusion string, d time.Duration) Run {
	start := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	return Run{Name: name, Conclusion: conclusion, StartedAt: start, UpdatedAt: start.Add(d)}
}

func TestPercentile(t *testing.T) {
	ds := []time.Duration{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cases := map[int]time.Duration{50: 5, 95: 10, 1: 1, 100: 10}
	for p, want := range cases {
		if got := percentile(ds, p); got != want {
			t.Errorf("p%d = %v, want %v", p, got, want)
		}
	}
}

func TestSummarize(t *testing.T) {
	runs := []Run{
		runAt("build", "success", 60*time.Second),
		runAt("build", "failure", 120*time.Second),
		runAt("lint", "success", 10*time.Second),
	}
	got := Summarize(runs)
	if len(got) != 2 || got[0].Workflow != "build" {
		t.Fatalf("want build first (slowest p95), got %+v", got)
	}
	b := got[0]
	if b.Runs != 2 || b.Avg != 90*time.Second || b.Max != 120*time.Second || b.SuccessRate != 0.5 {
		t.Errorf("unexpected build stats: %+v", b)
	}
}

func TestRunAgainstFakeAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/actions/runs" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"workflow_runs":[]}`)
			return
		}
		fmt.Fprint(w, `{"workflow_runs":[
			{"name":"ci","conclusion":"success","run_started_at":"2026-11-18T09:00:00Z","updated_at":"2026-11-18T09:01:30Z"},
			{"name":"ci","conclusion":"success","run_started_at":"2026-11-18T10:00:00Z","updated_at":"2026-11-18T10:00:30Z"}
		]}`)
	}))
	defer srv.Close()

	var out bytes.Buffer
	if err := run([]string{"-repo", "o/r", "-api", srv.URL}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ci") || !strings.Contains(out.String(), "1m0s") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
}

func TestRunRequiresRepo(t *testing.T) {
	if err := run([]string{}, &bytes.Buffer{}); err == nil {
		t.Error("expected error without -repo")
	}
}
