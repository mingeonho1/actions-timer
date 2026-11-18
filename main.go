// actions-timer prints how long each GitHub Actions workflow takes (avg / p50 / p95 / max).
// GitHub Actions 워크플로별 소요시간(평균·p50·p95·최대)과 성공률을 보여주는 CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("actions-timer", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name (required)")
	limit := fs.Int("limit", 100, "number of recent completed runs to analyze")
	asJSON := fs.Bool("json", false, "print JSON instead of a table")
	apiURL := fs.String("api", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		fs.Usage()
		return fmt.Errorf("-repo is required")
	}

	client := Client{
		BaseURL: *apiURL,
		Token:   os.Getenv("GITHUB_TOKEN"), // optional for public repos / 공개 저장소면 없어도 됨
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
	runs, err := client.FetchRuns(*repo, *limit)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		return fmt.Errorf("no completed runs found for %s", *repo)
	}

	stats := Summarize(runs)
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "WORKFLOW\tRUNS\tSUCCESS\tAVG\tP50\tP95\tMAX")
	for _, s := range stats {
		fmt.Fprintf(tw, "%s\t%d\t%.0f%%\t%s\t%s\t%s\t%s\n",
			s.Workflow, s.Runs, s.SuccessRate*100,
			short(s.Avg), short(s.P50), short(s.P95), short(s.Max))
	}
	return tw.Flush()
}

func short(d time.Duration) string {
	return d.Round(time.Second).String()
}
