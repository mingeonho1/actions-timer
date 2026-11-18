package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Run is the subset of a GitHub Actions workflow run we need.
// 워크플로 실행 1건에서 소요시간 계산에 필요한 필드만 받는다.
type Run struct {
	Name       string    `json:"name"`
	Conclusion string    `json:"conclusion"`
	StartedAt  time.Time `json:"run_started_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Duration is wall-clock time from start to the last update (≈ finish).
func (r Run) Duration() time.Duration {
	return r.UpdatedAt.Sub(r.StartedAt)
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// FetchRuns pages through completed runs until limit is reached or runs run out.
// 완료된 실행만 100개씩 페이지로 가져온다.
func (c Client) FetchRuns(repo string, limit int) ([]Run, error) {
	var runs []Run
	for page := 1; len(runs) < limit; page++ {
		url := fmt.Sprintf("%s/repos/%s/actions/runs?status=completed&per_page=100&page=%d", c.BaseURL, repo, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		var body struct {
			WorkflowRuns []Run `json:"workflow_runs"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API %s: %s", repo, resp.Status)
		}
		if err != nil {
			return nil, err
		}
		if len(body.WorkflowRuns) == 0 {
			break
		}
		runs = append(runs, body.WorkflowRuns...)
	}
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}
