# actions-timer

[![CI](https://github.com/mingeonho1/actions-timer/actions/workflows/ci.yml/badge.svg)](https://github.com/mingeonho1/actions-timer/actions/workflows/ci.yml)

A tiny Go CLI that shows how long each GitHub Actions workflow takes — average, p50, p95, max — and its success rate.
No dependencies beyond the Go standard library.

GitHub Actions 워크플로별 소요시간(평균·p50·p95·최대)과 성공률을 표로 보여주는 Go CLI입니다.
"어느 파이프라인이 제일 느린가, 느린 날은 얼마나 느린가(p95)"를 빠르게 보려고 만들었습니다.

## Usage

```bash
go install github.com/mingeonho1/actions-timer@latest

actions-timer -repo cli/cli -limit 200
GITHUB_TOKEN=... actions-timer -repo owner/private-repo   # token needed for private repos
actions-timer -repo owner/name -json                       # machine-readable output
```

```
WORKFLOW                    RUNS  SUCCESS  AVG     P50     P95     MAX
Unit and Integration Tests  25    76%      5m11s   6m19s   7m20s   7m53s
Lint                        18    67%      2m8s    2m41s   5m28s   5m28s
Code Scanning               25    76%      2m27s   2m55s   3m19s   5m27s
```

Rows are sorted by p95, slowest first. Duration = `updated_at − run_started_at` of each completed run.

## CI

The same pipeline is defined for two CI systems:

| | GitHub Actions (`.github/workflows/ci.yml`) | GitLab CI (`.gitlab-ci.yml`) |
|---|---|---|
| test | `go vet`, `go test -race` | `go vet`, `go test -race` + coverage parsing |
| build | single binary | cross-compiled linux/amd64 + darwin/arm64, kept as artifacts |
| measure | runs the tool on this repo, writes the table to the job summary | runs the built artifact on this repo |

## Layout

| File | Role |
|---|---|
| `main.go` | flags, table / JSON output |
| `github.go` | GitHub REST API client with pagination |
| `stats.go` | grouping, nearest-rank percentiles |
| `main_test.go` | unit tests + end-to-end test against a fake API (`httptest`) |

## License

MIT
