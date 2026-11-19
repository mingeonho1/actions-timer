# 실습 4. 파이프라인 속도 측정 → 개선 → 전후 비교

⏱ 40분 (+ CI 기다리는 시간) · 📍 로컬(Mac) + GitHub · 선행: 실습 1

## 목표

"CI를 빠르게 했다"는 말은 **숫자**가 있어야 믿을 수 있습니다. 이 도구(actions-timer)로 이 저장소 자신의 CI를 재고, 하나를 바꾸고, 다시 재서 비교합니다.

1. 비교가 공정하도록 **측정 범위를 거르는 플래그** 두 개(`-created`, `-event`)를 CLI에 추가
2. **기준선(baseline)** 숫자 확보
3. CI 개선: 잡 사이의 불필요한 직렬 의존 끊기 + 중복 실행 취소(`concurrency`)
4. 개선 후 숫자와 비교하고, **트레이드오프**까지 정리

> 💬 면접에서 이렇게 말할 수 있다:
> "직접 만든 CLI로 CI 소요시간 p50/p95를 재 보니, 측정 잡이 테스트 잡을 불필요하게 기다리고 있었습니다. 의존을 끊어 병렬로 돌리고 전후를 같은 조건(push 이벤트, 같은 기간 필터)으로 비교했습니다. 대신 테스트가 실패해도 측정 잡이 도는 비용이 생겨서 그 트레이드오프도 정리했습니다."

## 왜 필터가 먼저 필요한가

지금 이 저장소의 `CI` 워크플로를 그냥 재면 **두 종류의 실행이 섞여** 있습니다.

```bash
go run . -repo mingeonho1/actions-timer -workflow CI
```

`.github/workflows/ci.yml` 을 보면:

- `push`(main) 이벤트 → `test` 다음에 `measure` 까지 돈다 → 길다
- `pull_request` 이벤트 → `test` 만 돈다 (`measure` 에 `if: github.event_name == 'push'`) → 짧다

둘을 섞어 평균을 내면, PR이 많은 주엔 "빨라진 것처럼" 보입니다. **측정 대상이 섞이면 개선 효과를 말할 수 없습니다.** 그래서 이벤트와 기간을 거르는 플래그를 먼저 만듭니다.

## 준비

```bash
cd ~/github/actions-timer
git checkout main && git pull
git checkout -b lab4
```

실습 1의 `-workflow` 가 `main` 에 있어야 합니다. 아직 머지 전이면 실습 1 브랜치에서 이어서 해도 됩니다.

## 따라하기

### 1단계. `-created`, `-event` 플래그 추가 (15분)

GitHub API(`GET /repos/{owner}/{repo}/actions/runs`)는 쿼리 파라미터로 서버에서 걸러 줍니다.

| 파라미터 | 예 | 뜻 |
|---|---|---|
| `event` | `push` | 이 이벤트로 시작된 실행만 |
| `created` | `>=2026-11-18T12:00:00Z`, `<2026-11-18T12:00:00Z` | 생성 시각 범위 |

할 일:

1. `github.go` 의 `Client` 에 `Event`, `Created` 문자열 필드 추가
2. `FetchRuns` 에서 URL을 만들 때, 값이 있으면 `event=`, `created=` 를 붙이기
3. `main.go` 에 `-event`, `-created` 플래그를 만들고 `Client` 에 넘기기

**함정**: `created` 값의 `>`, `=`, `:` 는 URL에 그대로 넣으면 안 됩니다. `fmt.Sprintf` 로 문자열을 이어 붙이지 말고 **`net/url` 의 `url.Values` 로 만들고 `Encode()`** 하세요. 그러면 `>=` 가 `%3E%3D` 로 바뀝니다.

또 하나: 지금 `FetchRuns` 안에 `url := fmt.Sprintf(...)` 라는 지역 변수가 있습니다. `net/url` 패키지와 이름이 같아서, 그 줄 **아래에서** `url.Something` 을 쓰면 패키지 대신 문자열 변수로 읽힙니다. 헷갈리지 않게 변수 이름을 `endpoint` 같은 걸로 바꾸세요.

테스트(`query_test.go`): 가짜 서버가 **받은 쿼리 값**을 기록해 두고, 기대값과 비교합니다.

```go
package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatedAndEventAreSentToAPI(t *testing.T) {
	var gotCreated, gotEvent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCreated = r.URL.Query().Get("created") // Query()가 %3E%3D 를 >= 로 되돌려 준다
		gotEvent = r.URL.Query().Get("event")
		if r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"workflow_runs":[]}`)
			return
		}
		fmt.Fprint(w, `{"workflow_runs":[
			{"name":"ci","conclusion":"success","run_started_at":"2026-11-18T09:00:00Z","updated_at":"2026-11-18T09:01:00Z"}
		]}`)
	}))
	defer srv.Close()

	args := []string{"-repo", "o/r", "-api", srv.URL, "-created", ">=2026-11-18T12:00:00Z", "-event", "push"}
	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if gotCreated != ">=2026-11-18T12:00:00Z" || gotEvent != "push" {
		t.Errorf("created = %q, event = %q", gotCreated, gotEvent)
	}
}
```

```bash
gofmt -l . && go vet ./... && go test ./...
go run . -repo mingeonho1/actions-timer -workflow CI -event push
go run . -repo mingeonho1/actions-timer -workflow CI -event pull_request
```

두 결과의 AVG가 눈에 띄게 다르면, 섞여 있던 게 맞다는 증거입니다.

이 코드를 PR로 올려 `main` 에 머지해 둡니다(실습 1과 같은 방법).

### 2단계. 기준선 만들기 (5분 + 기다림)

비교하려면 각 쪽에 실행이 최소 3~5개는 있어야 합니다. 빈 커밋으로 `main` 에 push 해서 CI를 돌립니다. **한 번 push 하고, 그 실행이 끝난 뒤에 다음 push** 를 하세요(겹치면 나중 단계의 `concurrency` 실험과 섞입니다).

```bash
git checkout main && git pull
git commit --allow-empty -m "ci: baseline run 1"
git push
gh run watch --exit-status   # 끝날 때까지 기다림. 2, 3번도 반복
```

> 개인 실습 저장소라 `main` 에 직접 빈 커밋을 push 합니다. 회사 저장소였다면 `workflow_dispatch` 트리거로 수동 실행하는 게 맞습니다.

기준선이 다 모였으면 **지금 시각(UTC)** 을 기록해 둡니다. 이 시각이 전/후를 가르는 선입니다.

```bash
T0=$(date -u +%Y-%m-%dT%H:%M:%SZ); echo $T0
```

```bash
go run . -repo mingeonho1/actions-timer -workflow CI -event push -created "<$T0"
```

이 표를 그대로 복사해 둡니다. **기준선**입니다.

### 3단계. 왜 느린지 본다 (5분)

GitHub에서 최근 `CI` 실행 하나를 열면 잡 그림이 `test → measure` 로 이어져 있습니다. 워크플로 전체 시간 ≈ `test` 시간 + `measure` 시간입니다.

질문: `measure` 는 **`test` 의 결과물이 필요한가요?** `measure` 는 소스를 다시 받아 `go run` 으로 GitHub API를 부를 뿐, `test` 가 만든 파일을 쓰지 않습니다. 그런데 `needs: test` 때문에 기다립니다. 이게 개선 포인트입니다.

### 4단계. 개선한다 (5분)

`.github/workflows/ci.yml` 을 두 군데 고칩니다.

1. `measure` 잡의 `needs: test` 줄 삭제 → 두 잡이 **동시에** 시작
2. 워크플로 최상위에 `concurrency` 추가 → 같은 브랜치에 새 커밋이 오면 **이전 실행을 취소**해서 러너 낭비를 줄임

```yaml
# 같은 브랜치에 새 커밋이 오면 이전 실행은 취소한다.
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

`on:` 블록과 `jobs:` 사이에 넣습니다.

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('yaml ok')"
git checkout -b lab4-ci
git add .github/workflows/ci.yml
git commit -m "ci: run measure in parallel with test, cancel superseded runs"
git push -u origin lab4-ci
gh pr create --fill
```

머지합니다. 머지 push 자체가 "개선 후" 첫 실행이 됩니다.

### 5단계. 개선 후 측정 (5분 + 기다림)

2단계처럼 빈 커밋을 2~4번 더 push 해서 개선 후 실행을 3~5개 모읍니다. 그리고:

```bash
go run . -repo mingeonho1/actions-timer -workflow CI -event push -created ">=$T0"
```

기준선 표와 나란히 놓고 P50, P95를 비교합니다. 같은 셸이 아니라 `T0` 가 사라졌다면 2단계에서 적어 둔 값을 직접 넣으세요.

### 6단계. 결과를 한 문단으로 정리 (5분)

README나 노트에 이런 식으로 남겨 두세요. 숫자는 실제 측정값으로 바꿉니다.

```markdown
## CI 개선 기록
- 측정: actions-timer, workflow=CI, event=push, 전 N회 / 후 M회
- 변경: measure 잡의 needs: test 제거(병렬화), concurrency로 중복 실행 취소
- 결과: p50 1m15s → 0m45s, p95 1m33s → 0m50s
- 트레이드오프: test가 실패해도 measure가 끝까지 돈다(러너 시간 약간 낭비).
  measure는 배포가 아니라 조회라서 받아들임. 배포 잡이었다면 needs를 유지했을 것.
```

## 확인 방법 (이게 보이면 성공)

- `go test ./...` 통과, `-event push` 와 `-event pull_request` 결과가 다르게 나온다
- 개선 후 실행 화면에서 `test` 와 `measure` 가 **동시에** 시작한다 (잡 그림에 연결선이 없다)
- 전/후 표 두 개와 위 정리 문단이 있다

## 자주 막히는 곳

- **`-created "<..."` 에서 셸 에러가 나요**: `<`, `>` 는 셸의 리다이렉션 기호입니다. 반드시 **따옴표로** 감싸세요. 따옴표 없이 쓰면 zsh는 `... not found` 에러를 내고, bash는 그 이름의 파일을 만들어 버립니다.
- **결과가 0건(`no completed runs found`)**: 시각은 **UTC**, 형식은 `2026-11-18T12:00:00Z` 입니다. 한국 시간으로 쓰면 9시간 어긋납니다.
- **숫자가 오히려 늘었어요**: 실행 수가 적으면 러너 대기·네트워크 같은 잡음이 큽니다. 실행을 더 모으고, 평균보다 p50을 보세요. 그래서 이 도구가 평균 말고 p50/p95를 같이 보여 줍니다.
- **`url` 이름 충돌**: `url.Values undefined (type string has no field or method Values)` 가 나면, 함수 안의 `url := ...` 지역 변수가 `net/url` 패키지 이름을 가린 겁니다. 변수 이름을 바꾸세요.
- **concurrency 때문에 실행이 `Cancelled` 로 남아요**: 빈 커밋을 연달아 push 하면 앞의 것이 취소됩니다. 의도한 동작입니다. 측정에는 `status=completed` 로 들어오지만 conclusion이 `cancelled` 라 SUCCESS 비율이 내려갑니다. 측정할 땐 하나씩 기다렸다 push 하세요.

## 더 해보기 (선택)

- `ci.yml` 의 `measure` 잡 명령을 `-workflow CI -event push` 로 바꿔, 실행 요약 페이지에 깨끗한 숫자만 남게 해 보세요.
- 실습 2의 GitLab 파이프라인에서도 같은 질문을 해 보세요: `measure` 는 `build` 의 artifact가 **필요**하니 `needs: [build]` 를 끊으면 안 됩니다. "끊어도 되는 의존"과 "끊으면 안 되는 의존"을 구분하는 게 이 실습의 핵심입니다.

<details>
<summary>정답 예시 (먼저 직접 해 보고 여세요)</summary>

`github.go` — import와 `Client`:

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
	Event   string // "push", "pull_request" ... / 비어 있으면 필터 없음
	Created string // GitHub "created" filter, e.g. ">=2026-11-18" / 비어 있으면 필터 없음
}
```

`github.go` — `FetchRuns` 반복문 앞부분:

```go
	for page := 1; len(runs) < limit; page++ {
		q := url.Values{}
		q.Set("status", "completed")
		q.Set("per_page", "100")
		q.Set("page", strconv.Itoa(page))
		if c.Created != "" {
			q.Set("created", c.Created)
		}
		if c.Event != "" {
			q.Set("event", c.Event)
		}
		endpoint := fmt.Sprintf("%s/repos/%s/actions/runs?%s", c.BaseURL, repo, q.Encode())
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
```

`main.go` — 플래그와 `Client`:

```go
	created := fs.String("created", "", `filter runs by creation time, e.g. ">=2026-11-18T12:00:00Z"`)
	event := fs.String("event", "", "only runs triggered by this event, e.g. push or pull_request")
```

```go
	client := Client{
		BaseURL: *apiURL,
		Token:   os.Getenv("GITHUB_TOKEN"), // optional for public repos / 공개 저장소면 없어도 됨
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Created: *created,
		Event:   *event,
	}
```

`.github/workflows/ci.yml` — 바뀐 부분만:

```yaml
on:
  push:
    branches: [main]
  pull_request:

# 같은 브랜치에 새 커밋이 오면 이전 실행은 취소한다.
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test:
    # (그대로)

  measure:
    # needs: test  ← 삭제
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    # (나머지 그대로)
```

</details>
