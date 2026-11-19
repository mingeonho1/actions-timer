# actions-timer로 배우는 CI/CD 실습

이 저장소(작은 Go CLI + GitHub Actions + GitLab CI)를 직접 고치면서 CI/CD 개념을 손으로 익히는 실습 모음입니다.
실습마다 20~40분이면 끝나고, Mac에 **Go, git, 브라우저, GitHub·GitLab 계정**만 있으면 됩니다. Kubernetes·Helm·ArgoCD는 쓰지 않습니다.

## 실습 목록

| # | 실습 | 시간 | 배우는 것 | 공고와의 연결 |
|---|---|---|---|---|
| 1 | [Go CLI에 `-workflow` 필터 추가하기](01-go-cli-feature.md) | 30분 | 플래그 추가, table-driven test, `httptest` 가짜 서버 | "CLI 도구 기능 개발 참여 (Go)" |
| 2 | [GitLab CI 문법 손으로 익히기](02-gitlab-ci.md) | 40분 | `stages` vs `needs`(DAG), `rules`, cache vs artifacts | "표준 CI/CD 파이프라인 개선 (GitLab CI)" |
| 3 | [태그 push로 릴리스 자동화](03-release-automation.md) | 30분 | 태그 트리거, 크로스 컴파일, `-ldflags`로 버전 주입, `permissions` | "빌드·테스트·배포 파이프라인 직접 구성" (우대) |
| 4 | [파이프라인 속도 측정 → 개선 → 전후 비교](04-measure-and-optimize.md) | 40분 | 측정 기준 잡기, `concurrency`, 잡 의존 끊기, 데이터 오염 거르기 | "표준 CI/CD 파이프라인 개선", "배포 Workflow 시스템" |

## 오늘 밤 추천 순서

1. **실습 1** (필수) — 실습 4에서 이 기능을 그대로 씁니다.
2. **실습 2** (필수) — 공고에 GitLab CI가 이름까지 적혀 있습니다. 시간이 없으면 이것부터 해도 됩니다.
3. **실습 3** — 시간이 남으면. 태그 하나로 바이너리가 Release에 붙는 걸 보면 "배포 자동화"가 감이 옵니다.
4. **실습 4** — 다른 날 해도 됩니다. CI가 여러 번 돌아야 숫자가 모여서, 중간중간 기다리는 시간이 있습니다.

## 시작 전에 한 번만

```bash
export PATH=$HOME/.local/go/bin:$PATH   # Go가 여기 깔려 있다면
cd ~/github/actions-timer
git checkout main && git pull
go version
go test ./...
```

`ok  github.com/mingeonho1/actions-timer` 가 보이면 준비 끝입니다.

## 규칙

- 실습마다 브랜치를 새로 만듭니다: `git checkout -b lab1` 처럼.
- 고치고 나면 항상 아래 세 줄을 돌립니다. CI가 하는 일을 내 손으로 먼저 해 보는 습관입니다.

```bash
gofmt -l .        # 아무것도 안 나와야 함
go vet ./...      # 아무것도 안 나와야 함
go test ./...     # ok 가 나와야 함
```

- 정답은 각 실습 끝의 `정답 예시` 접기 안에 있습니다. 먼저 혼자 해 보고 막히면 여세요.
- 각 실습 맨 위의 "면접에서 이렇게 말할 수 있다" 한 줄은, 끝내고 나서 내 말로 바꿔 적어 두세요.
