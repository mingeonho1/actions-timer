# 실습 2. GitLab CI 문법 손으로 익히기

⏱ 40분 · 📍 브라우저(GitLab 웹)만

## 목표

`.gitlab-ci.yml` 을 고쳐 가며 아래 세 가지를 **눈으로** 확인합니다.

1. **`stages` vs `needs`**: 단계(stage) 순서대로 도는 파이프라인을, `needs` 로 의존 관계(DAG)만 따라 도는 파이프라인으로 바꾸기
2. **`rules`**: 브랜치에 따라 잡을 켜고 끄고, 수동 실행(`when: manual`)으로 바꾸기
3. **cache vs artifacts**: 둘 다 "파일을 다음으로 넘기는 것"처럼 보이지만 목적이 다르다는 걸, 일부러 망가뜨려 보며 확인

> 💬 면접에서 이렇게 말할 수 있다:
> "GitLab CI에서 `needs` 로 DAG를 구성해 단계 대기 시간을 줄였고, 잡 사이에 반드시 넘겨야 하는 빌드 결과물은 artifacts로, 있으면 빨라지는 의존성은 cache로 구분해서 썼습니다. artifacts를 빼면 다음 잡이 실제로 깨지는 것도 확인했습니다."

## 준비

- GitLab 프로젝트: <https://gitlab.com/mingeonho1/actions-timer>
- 이 실습은 로컬 push 없이 **GitLab 웹의 Pipeline editor** 로만 합니다. (GitLab 쪽은 GitHub에서 한 번 가져온 복사본이라, 여기서 고쳐도 GitHub에는 영향이 없습니다.)

1. 왼쪽 메뉴 **Code → Branches → New branch**
   - Branch name: `lab2`, Create from: `main` → **Create branch**
2. 왼쪽 메뉴 **Build → Pipeline editor**
   - 화면 위쪽 브랜치 선택에서 **`lab2`** 로 바꿉니다. (실수로 `main` 에 커밋하지 않도록!)

Pipeline editor에는 탭이 세 개 있습니다. 오늘 계속 씁니다.

| 탭 | 쓰는 때 |
|---|---|
| **Edit** | YAML 고치기. 문법이 틀리면 위에 바로 빨간 경고가 뜹니다 |
| **Visualize** | 잡들이 어떻게 연결되는지 그림으로 보기 |
| **Validate** | "이 브랜치에 push 하면 어떤 잡이 생기는지" 시뮬레이션 |

## 따라하기

### 1단계. 지금 파이프라인을 그림으로 본다 (5분)

**Visualize** 탭을 엽니다. `test → build → measure` 세 칸이 차례로 보입니다.

**Build → Pipelines** 에서 최근 `main` 파이프라인 하나를 열고, 잡 이름 옆 시간을 적어 두세요. 나중에 비교합니다.

생각해 볼 질문: `build` 는 정말 `test` 가 끝날 때까지 기다려야 할까요? 테스트가 실패하면 어차피 배포는 안 하니 맞는 말이기도 하고, 빌드가 되는지는 테스트와 상관없이 빨리 알고 싶기도 합니다. 정답은 팀마다 다릅니다. 오늘은 "기다리지 않는" 쪽을 만들어 봅니다.

### 2단계. `lint` 잡 추가 + `needs` 로 DAG 만들기 (10분)

**Edit** 탭에서 `test:` 잡 아래에 `lint` 잡을 추가합니다. 같은 `test` 단계라서 `test` 잡과 **동시에** 돕니다.

```yaml
lint:
  stage: test
  script:
    - gofmt -l *.go | tee gofmt.out
    - test ! -s gofmt.out   # 포맷 안 된 파일이 하나라도 있으면 실패
```

`gofmt -l` 은 포맷이 틀린 파일 **이름만** 출력하고 종료 코드는 항상 0입니다. 그래서 "출력이 비어 있지 않으면 실패"를 `test ! -s` 로 직접 만들어 줍니다. (`-s`: 파일 크기가 0보다 크다)

이제 `build` 잡에 한 줄을 넣습니다.

```yaml
build:
  stage: build
  needs: [lint]            # test를 기다리지 않고 lint만 끝나면 바로 시작
  script:
    ...
```

**Visualize** 탭을 다시 봅니다. 이번엔 **Job dependencies** 토글(또는 "Show dependencies")을 켜면 `lint → build → measure` 선이 보이고, `test` 는 혼자 떨어져 있습니다.

아래 **Commit changes** 로 `lab2` 브랜치에 커밋합니다. **Build → Pipelines** 에 새 파이프라인이 생깁니다.

> `needs` 가 있는 잡은 **stage 순서를 무시하고** 지정한 잡만 끝나면 시작합니다. 그래서 `build` 는 `test` 가 아직 도는 중에도 시작할 수 있습니다.

### 3단계. `rules` 로 브랜치별 실행 다루기 (10분)

방금 파이프라인에 `measure` 잡이 **없을** 겁니다. `measure` 에 이런 규칙이 있기 때문입니다.

```yaml
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH   # main 일 때만
```

`rules` 는 **위에서부터 처음 맞는 규칙 하나**가 적용되고, 아무것도 안 맞으면 잡이 아예 만들어지지 않습니다.

이번엔 `build` 에 규칙을 줍니다. 목표:

- `main` → 자동 실행
- `lab2` → 자동 실행 (실습용)
- 그 밖의 브랜치 → 버튼을 눌러야 실행(수동), 안 눌러도 파이프라인은 통과

그리고 `measure` 도 `lab2` 에서 돌도록 규칙을 한 줄 추가합니다. (다음 단계에서 쓸 겁니다.)

직접 써 본 뒤, **Validate** 탭에서 브랜치를 `lab2` 로 두고 **Validate pipeline** 을 눌러 어떤 잡이 생기는지 확인합니다. 정답은 맨 아래 접기에 있습니다.

> 자주 쓰는 미리 정의된 변수: `$CI_COMMIT_BRANCH`(브랜치 이름), `$CI_DEFAULT_BRANCH`(보통 `main`), `$CI_COMMIT_TAG`(태그 push 때만 값이 있음), `$CI_PIPELINE_SOURCE`(`push`, `merge_request_event`, `web` …)

커밋하고, 새 파이프라인에 `test`, `lint`, `build`, `measure` 네 잡이 다 생기는지 봅니다.

### 4단계. artifacts를 일부러 빼서 망가뜨려 본다 (10분)

`build` 잡에서 `artifacts:` 블록 세 줄을 **지웁니다**(주석 처리해도 됩니다).

```yaml
  # artifacts:
  #   paths: [dist/]
  #   expire_in: 1 week
```

커밋 후 `measure` 잡 로그를 엽니다. 이런 줄이 보이고 잡이 실패합니다.

```
./dist/actions-timer-linux-amd64: No such file or directory
```

왜일까요? **잡마다 새 컨테이너에서 깨끗하게 시작**하기 때문입니다. `build` 가 만든 `dist/` 는 `build` 잡이 끝나면서 사라집니다. 다음 잡에 넘기려면 artifacts로 **명시적으로 업로드**해야 합니다.

`artifacts:` 를 되살리고 다시 커밋합니다. 이번엔 `measure` 로그 앞부분에 다음 줄이 보입니다.

```
Downloading artifacts for build (...)...
```

### 5단계. cache는 뭐가 다른가 (5분)

같은 `measure` 로그와 `test` 로그에서 **cache** 관련 줄도 찾아보세요.

```
Checking cache for ...
Restoring cache
...
Saving cache for successful job
```

이 저장소는 외부 의존성이 없어서(`go.sum` 없음) 캐시가 거의 비어 있습니다. 경고가 보여도 정상입니다.

둘의 차이를 표로 정리하면:

| | artifacts | cache |
|---|---|---|
| 목적 | 잡이 **만든 결과물**을 다음 잡·사람에게 넘김 | 다음 실행을 **빠르게** 하려고 재사용 |
| 없으면 | 다음 잡이 **깨진다** | 느려질 뿐 **결과는 같다** |
| 보장 | `needs`/stage 순서대로 반드시 전달 | 있을 수도, 없을 수도 있음(best-effort) |
| 예 | 빌드한 바이너리, 테스트 리포트, coverage | `go mod` 다운로드, `node_modules` |
| 다운로드 | 파이프라인 화면에서 내려받기 가능 | 사람이 내려받는 용도 아님 |

**한 문장 규칙**: "없으면 틀린 결과가 나오는 것은 artifacts, 없으면 느리기만 한 것은 cache."

## 확인 방법 (이게 보이면 성공)

- **Visualize** 에서 `lint → build → measure` 연결선이 보이고 `test` 는 따로 떨어져 있다
- `lab2` 파이프라인에 `test`, `lint`, `build`, `measure` 네 잡이 있고 모두 통과
- artifacts를 뺐을 때 `measure` 가 `No such file or directory` 로 실패했고, 되살리니 `Downloading artifacts` 와 함께 통과
- 다른 이름의 브랜치(예: `lab2-manual`)를 만들어 보면 `build` 가 ▶ 버튼(수동)으로 나오고 `measure` 는 없다

## 자주 막히는 곳

- **`'measure' job needs 'build' job, but 'build' was not added to the pipeline` 비슷한 에러**: `measure` 는 생기는데 `needs` 로 가리키는 `build` 가 `rules` 때문에 안 생긴 경우입니다. 두 잡의 `rules` 가 같은 브랜치를 다 커버하는지 확인하세요.
- **`when: manual` 잡 때문에 파이프라인이 "blocked"**: `allow_failure: true` 를 빼먹으면 수동 잡을 누르기 전까지 파이프라인이 끝나지 않습니다.
- **YAML 들여쓰기**: 탭 문자 금지, 공백 2칸. Edit 탭 위에 "This GitLab CI configuration is invalid" 가 뜨면 거의 들여쓰기 문제입니다.
- **`lint` 가 갑자기 실패**: `gofmt -l .` 로 쓰면 캐시 폴더(`.go/`)까지 뒤질 수 있어서 `*.go` 로 범위를 좁혔습니다.
- **실수로 `main` 에 커밋했다**: Pipeline editor 위쪽 브랜치를 항상 확인하세요. GitLab 쪽 `main` 은 GitHub와 별개라 GitHub에는 영향이 없지만, 배지(badge)가 빨개질 수 있습니다.

## 더 해보기 (선택)

- `test` 잡에 `parallel: matrix:` 로 Go 버전 두 개(`golang:1.24`, `golang:1.25`)를 동시에 돌려 보세요.
- `build` 의 artifacts 아래에 `when: always` / `expire_in: 1 hour` 를 바꿔 보고, 파이프라인 화면의 다운로드 버튼이 어떻게 달라지는지 보세요.
- 끝나면 `lab2` 브랜치는 **Code → Branches** 에서 지워도 됩니다.

<details>
<summary>정답 예시 — 완성된 <code>.gitlab-ci.yml</code> (lab2 브랜치용)</summary>

```yaml
# Same pipeline on GitLab CI: test -> build -> measure.
# GitHub Actions(ci.yml)와 같은 흐름을 GitLab CI 문법으로 옮긴 것.
image: golang:1.24

stages:
  - test
  - build
  - measure

variables:
  GOPATH: $CI_PROJECT_DIR/.go   # keep module cache inside the project so it can be cached

cache:
  key:
    files: [go.mod]
  paths: [.go/pkg/mod/]

test:
  stage: test
  script:
    - go vet ./...
    - go test -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out | tail -1
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'

lint:
  stage: test
  script:
    - gofmt -l *.go | tee gofmt.out
    - test ! -s gofmt.out   # 포맷 안 된 파일이 하나라도 있으면 실패

build:
  stage: build
  needs: [lint]            # test를 기다리지 않고 lint만 끝나면 바로 시작
  script:
    - CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/actions-timer-linux-amd64 .
    - CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/actions-timer-darwin-arm64 .
  artifacts:
    paths: [dist/]
    expire_in: 1 week
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_BRANCH == "lab2"
    - when: manual
      allow_failure: true

measure:
  stage: measure
  needs: [build]
  script:
    - ./dist/actions-timer-linux-amd64 -repo mingeonho1/actions-timer -limit 50
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_BRANCH == "lab2"
```

`build` 의 세 번째 규칙(`when: manual`)에는 `if` 가 없습니다. 위 두 개에 안 걸린 **나머지 전부**에 적용된다는 뜻입니다.

</details>
