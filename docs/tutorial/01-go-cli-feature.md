# 실습 1. Go CLI에 `-workflow` 필터 추가하기 (테스트 먼저)

⏱ 30분 · 📍 로컬(Mac) + GitHub

## 목표

- 지금은 저장소의 **모든 워크플로**가 표에 나옵니다. `-workflow CI` 처럼 이름을 주면 그 워크플로만 보이게 만듭니다.
- 코드를 쓰기 **전에** 테스트를 먼저 씁니다. Go에서 가장 흔한 두 가지 테스트 방식을 직접 씁니다.
  - **table-driven test**: 입력·기대값을 표(슬라이스)로 늘어놓고 반복문으로 검사
  - **`httptest`**: 진짜 GitHub 대신 가짜 HTTP 서버를 띄워 CLI 전체를 끝까지(end-to-end) 실행

> 💬 면접에서 이렇게 말할 수 있다:
> "사내 CLI에 기능을 더할 때처럼, 순수 함수는 table-driven test로, 외부 API 호출은 `httptest` 가짜 서버로 테스트해서 네트워크 없이도 CI에서 결정적으로 도는 테스트를 만들었습니다."

## 준비

```bash
cd ~/github/actions-timer
git checkout main && git pull
git checkout -b lab1
```

코드 흐름을 먼저 5분만 읽습니다.

| 파일 | 볼 곳 | 하는 일 |
|---|---|---|
| `main.go` | `run()` | 플래그 파싱 → `FetchRuns` → `Summarize` → 표 출력 |
| `github.go` | `FetchRuns()` | GitHub API에서 완료된 실행을 페이지 단위로 가져옴 |
| `stats.go` | `Summarize()` | 워크플로 이름별로 묶어 avg/p50/p95/max 계산 |
| `main_test.go` | `runAt()`, `TestRunAgainstFakeAPI` | 테스트용 `Run` 만드는 헬퍼, 가짜 API 서버 예시 |

**어디에 끼워 넣을지**: `FetchRuns`로 가져온 `runs`를 `Summarize`에 넘기기 **직전에** 이름으로 거르면 됩니다.

## 따라하기

### 1단계. 실패하는 테스트부터 쓴다 (Red)

새 파일 `filter_test.go` 를 만듭니다. 아직 없는 함수 `filterRuns` 를 부르는 테스트입니다.

```go
package main

import (
	"testing"
	"time"
)

func TestFilterRuns(t *testing.T) {
	runs := []Run{
		runAt("CI", "success", time.Minute),
		runAt("Release", "success", 2*time.Minute),
		runAt("CI", "failure", 3*time.Minute),
	}

	tests := []struct {
		name   string // 케이스 설명 (go test -v 에 그대로 찍힌다)
		filter string
		want   int
	}{
		{"빈 필터면 전부", "", 3},
		{"정확히 일치하는 것만", "CI", 2},
		{"없는 이름이면 0개", "Deploy", 0},
		{"대소문자는 구분", "ci", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterRuns(runs, tt.filter)
			if len(got) != tt.want {
				t.Errorf("filterRuns(%q) = %d runs, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}
```

```bash
go test ./...
```

`undefined: filterRuns` 로 **컴파일 에러**가 나면 정상입니다. 이게 "Red" 상태입니다.

### 2단계. 테스트를 통과시키는 최소 코드 (Green)

`main.go` 맨 아래에 `filterRuns` 를 직접 써 보세요. 규칙은 테스트 표에 다 있습니다.

- 빈 문자열이면 받은 그대로 돌려준다
- 아니면 `r.Name == name` 인 것만 모은다

```bash
go test -run TestFilterRuns -v ./...
```

네 줄 모두 `--- PASS` 면 성공입니다. 케이스 이름이 한글로 찍히는 것도 확인하세요(공백은 `_` 로 바뀝니다).

### 3단계. 플래그를 연결한다

`main.go` 의 `run()` 에서 두 군데를 고칩니다.

1. 플래그 정의 근처에 `-workflow` 문자열 플래그를 추가 (`fs.String(...)`)
2. `FetchRuns` 다음, `len(runs) == 0` 검사 **앞에서** `runs = filterRuns(runs, *workflow)`

검사 앞에 넣는 이유: 이름이 하나도 안 맞으면 기존 에러(`no completed runs found`)가 그대로 나가게 하려고요.

### 4단계. CLI 전체를 끝까지 테스트한다 (`httptest`)

`filter_test.go` 에 테스트를 하나 더 붙입니다. 가짜 서버가 `ci`, `lint` 두 워크플로를 돌려주고, `-workflow ci` 로 실행했을 때 `lint` 가 **안** 보여야 합니다.

```go
func TestRunWorkflowFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"workflow_runs":[]}`) // 2페이지부터는 비어 있음 → 페이지 반복 종료
			return
		}
		fmt.Fprint(w, `{"workflow_runs":[
			{"name":"ci","conclusion":"success","run_started_at":"2026-11-18T09:00:00Z","updated_at":"2026-11-18T09:01:00Z"},
			{"name":"lint","conclusion":"success","run_started_at":"2026-11-18T09:00:00Z","updated_at":"2026-11-18T09:00:10Z"}
		]}`)
	}))
	defer srv.Close()

	var out bytes.Buffer
	if err := run([]string{"-repo", "o/r", "-api", srv.URL, "-workflow", "ci"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ci") || strings.Contains(out.String(), "lint") {
		t.Errorf("want only ci, got:\n%s", out.String())
	}
}
```

import 블록도 늘려야 합니다: `bytes`, `fmt`, `net/http`, `net/http/httptest`, `strings`.

핵심은 `-api srv.URL` 입니다. CLI가 원래 API 주소를 플래그로 받게 만들어 둔 덕분에, 코드 수정 없이 진짜 GitHub 대신 가짜 서버를 꽂을 수 있습니다. **테스트하기 쉬운 설계 = 바깥 의존성을 주입할 수 있는 설계**라는 걸 기억해 두세요.

### 5단계. 로컬 CI를 돌리고, 진짜 저장소에도 써 본다

```bash
gofmt -l . && go vet ./... && go test -race -cover ./...
go run . -repo mingeonho1/actions-timer -workflow CI
go run . -repo mingeonho1/actions-timer -workflow nope; echo "exit=$?"
go run . -repo cli/cli -limit 200 -workflow Lint
```

### 6단계. PR을 올리고 CI가 도는 걸 본다

```bash
git add main.go filter_test.go
git commit -m "feat: add -workflow flag to show a single workflow"
git push -u origin lab1
gh pr create --fill
```

PR 화면 아래 **Checks** 에서 `CI / test` 가 초록불이 되는지 봅니다. 내가 로컬에서 돌린 `go vet`, `go test -race` 를 CI가 똑같이 다시 돌리는 겁니다.

## 확인 방법 (이게 보이면 성공)

- `go test -run TestFilterRuns -v ./...` → 서브테스트 4개 모두 `PASS`
- `go run . -repo mingeonho1/actions-timer -workflow CI` → 표에 `CI` 한 줄만

```
WORKFLOW  RUNS  SUCCESS  AVG   P50   P95    MAX
CI        4     100%     1m9s  1m1s  1m33s  1m33s
```

(숫자는 그날그날 다릅니다.)

- `-workflow nope` → `error: no completed runs found for ...`, `exit=1`
- PR의 `CI / test` 체크 통과

## 자주 막히는 곳

- **`-workflow ci` 로 했는데 아무것도 안 나와요.** 이 저장소의 워크플로 이름은 `CI`(대문자)입니다. 정확히 일치로 만들었으니 정상 동작입니다. 이름은 `.github/workflows/ci.yml` 의 `name:` 값입니다.
- **`-limit 100 -workflow X` 인데 RUNS가 100보다 훨씬 적어요.** 최근 100개를 **먼저** 가져오고 **그다음** 거르기 때문입니다. 서버에서 거르려면 API의 `/actions/workflows/{id}/runs` 를 써야 하는데, 오늘은 범위 밖입니다. 면접에서 "트레이드오프를 알고 있다"고 말하기 좋은 지점입니다.
- **`imported and not used`** 에러. import 를 넣고 안 쓰면 Go는 컴파일을 거부합니다. 테스트를 다 쓴 뒤에 import 를 정리하세요.
- **gofmt가 파일 이름을 출력해요.** `gofmt -w .` 로 고친 뒤 다시 확인하세요.

## 더 해보기 (선택)

- 정확히 일치 대신 부분 일치(`strings.Contains`)로 바꾸고, 테스트 표에 케이스를 추가해 보세요. 표에 한 줄만 더하면 되는 게 table-driven test의 장점입니다.
- `-workflow` 에 아무것도 안 맞을 때 "사용 가능한 이름: CI, Release" 처럼 힌트를 주는 에러 메시지를 만들어 보세요.

<details>
<summary>정답 예시 (먼저 직접 해 보고 여세요)</summary>

`main.go` — 플래그 추가:

```go
	apiURL := fs.String("api", "https://api.github.com", "GitHub API base URL")
	workflow := fs.String("workflow", "", "only show this workflow (exact name)")
```

`main.go` — `FetchRuns` 다음:

```go
	runs, err := client.FetchRuns(*repo, *limit)
	if err != nil {
		return err
	}
	runs = filterRuns(runs, *workflow)
	if len(runs) == 0 {
		return fmt.Errorf("no completed runs found for %s", *repo)
	}
```

`main.go` — 맨 아래:

```go
// filterRuns keeps only runs whose workflow name matches exactly. Empty name keeps all.
// 이름이 정확히 같은 워크플로만 남긴다. 빈 문자열이면 전부 그대로.
func filterRuns(runs []Run, name string) []Run {
	if name == "" {
		return runs
	}
	var out []Run
	for _, r := range runs {
		if r.Name == name {
			out = append(out, r)
		}
	}
	return out
}
```

`filter_test.go` 의 import 블록:

```go
import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)
```

</details>
