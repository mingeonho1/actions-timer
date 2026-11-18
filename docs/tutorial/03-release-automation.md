# 실습 3. 태그 push로 릴리스 자동화하기

⏱ 30분 · 📍 로컬(Mac) + GitHub

## 목표

`git push origin v0.1.0` **한 번**으로 이런 일이 자동으로 일어나게 만듭니다.

1. 테스트를 돌리고
2. linux/macOS/windows용 바이너리 4개를 크로스 컴파일하고
3. 각 바이너리에 버전(`v0.1.0`)을 박아 넣고
4. GitHub **Releases** 페이지에 바이너리와 체크섬을 첨부

배우는 개념: **태그 트리거**, **크로스 컴파일**(`GOOS`/`GOARCH`), **빌드 시 값 주입**(`-ldflags -X`), **토큰 권한**(`permissions`), **체크섬**.

> 💬 면접에서 이렇게 말할 수 있다:
> "태그 push를 트리거로 테스트 → 4개 플랫폼 크로스 컴파일 → 버전 주입 → 체크섬 생성 → Release 첨부까지 자동화했습니다. 워크플로 토큰 권한은 필요한 `contents: write` 만 열었습니다."

## 준비

```bash
cd ~/github/actions-timer
git checkout main && git pull
git checkout -b lab3
```

## 따라하기

### 1단계. 바이너리가 자기 버전을 말하게 한다 (10분)

릴리스한 바이너리를 받은 사람이 "이게 몇 버전이지?"를 알 수 있어야 합니다. 버전을 코드에 직접 적으면 매번 고쳐야 하니, **빌드할 때 밖에서 주입**합니다.

`main.go` 에 패키지 변수를 하나 둡니다.

```go
// version is overwritten at build time: go build -ldflags "-X main.version=v0.1.0"
var version = "dev"
```

그리고 `run()` 에 `-version` 불리언 플래그를 추가해서, 켜져 있으면 `version` 을 출력하고 바로 `return nil` 하게 만드세요. **`-repo` 필수 검사보다 앞에** 있어야 `-repo` 없이도 동작합니다.

테스트도 하나 씁니다(`version_test.go`).

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-version"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "dev" {
		t.Errorf("version = %q, want dev", out.String())
	}
}
```

확인:

```bash
go test ./...
go run . -version
go build -ldflags "-X main.version=v9.9.9" -o /tmp/at . && /tmp/at -version
```

첫 번째는 `dev`, 두 번째는 `v9.9.9` 가 나오면 성공입니다.

> `-X main.version=...` 은 **패키지 경로.변수명** 형식입니다. `package main` 이라 `main.version`. 변수는 `var` 로 선언된 **문자열**이어야 합니다(`const` 는 안 됨).

### 2단계. 크로스 컴파일을 로컬에서 먼저 해 본다 (5분)

CI에 넣기 전에 같은 스크립트를 내 Mac에서 돌려 봅니다. Go는 `GOOS`, `GOARCH` 환경 변수만 바꾸면 다른 OS용 바이너리가 나옵니다.

```bash
GITHUB_REF_NAME=v0.0.0-local bash -c '
mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
  os=${target%/*}
  arch=${target#*/}
  ext=""
  if [ "$os" = windows ]; then ext=".exe"; fi
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
    -ldflags "-s -w -X main.version=${GITHUB_REF_NAME}" \
    -o "dist/actions-timer-${os}-${arch}${ext}" .
done
cd dist && shasum -a 256 * > checksums.txt
'
ls dist
file dist/*
./dist/actions-timer-darwin-arm64 -version
rm -rf dist
```

- `${target%/*}` 는 `/` 뒤를 잘라 `linux`, `${target#*/}` 는 앞을 잘라 `amd64` 를 얻는 셸 문법입니다.
- `CGO_ENABLED=0` 은 C 라이브러리 없이 순수 Go로 빌드 → 어느 리눅스에서든 그냥 도는 정적 바이너리.
- `-s -w` 는 디버그 정보를 빼서 파일 크기를 줄입니다.
- `file` 출력에 `ELF`(리눅스), `Mach-O`(맥), `PE32+`(윈도우)가 섞여 보이면 성공입니다.

### 3단계. Release 워크플로 작성 (10분)

`.github/workflows/release.yml` 을 새로 만듭니다. 뼈대만 주고, 빈칸은 직접 채워 보세요.

```yaml
name: Release

on:
  push:
    tags: ['v*']   # v0.1.0 같은 태그를 push 할 때만 실행

permissions:
  contents: ???    # Release를 만들려면?

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: false
      - name: Test
        run: go test ./...
      - name: Cross-compile
        run: |
          # 2단계 스크립트를 여기에 (단, 리눅스에는 shasum 대신 sha256sum)
      - name: Create GitHub Release
        env:
          GH_TOKEN: ${{ github.token }}
        run: gh release create ??? dist/* --title ??? --generate-notes
```

힌트:

- 태그 이름은 `GITHUB_REF_NAME` 환경 변수에 들어 있습니다(예: `v0.1.0`).
- GitHub가 만든 러너(`ubuntu-latest`)에는 `gh` CLI가 이미 깔려 있고, `GH_TOKEN` 만 주면 로그인 없이 씁니다.
- `--generate-notes` 는 지난 태그 이후 머지된 PR 목록으로 릴리스 노트를 자동 작성합니다.

### 4단계. 머지하고 태그를 찍는다 (5분)

워크플로 파일이 **태그가 가리키는 커밋 안에** 있어야 실행됩니다. 그래서 먼저 `main` 에 넣고 태그를 찍습니다.

```bash
gofmt -l . && go vet ./... && go test ./...
git add main.go version_test.go .github/workflows/release.yml
git commit -m "ci: release binaries on tag push"
git push -u origin lab3
gh pr create --fill
```

PR 체크가 초록이면 GitHub에서 머지한 뒤:

```bash
git checkout main && git pull
git tag v0.1.0
git push origin v0.1.0
```

```bash
gh run watch          # 방금 시작된 Release 실행을 터미널에서 따라가기
gh release view v0.1.0
```

### 5단계. 받아서 실행해 본다

```bash
mkdir -p /tmp/rel && cd /tmp/rel
gh release download v0.1.0 -R mingeonho1/actions-timer
shasum -a 256 -c checksums.txt --ignore-missing
chmod +x actions-timer-darwin-arm64
./actions-timer-darwin-arm64 -version
cd ~/github/actions-timer
```

## 확인 방법 (이게 보이면 성공)

- GitHub **Actions** 탭에 `Release` 실행이 생기고 초록불
- **Releases** 페이지에 `v0.1.0` 과 파일 5개(바이너리 4 + `checksums.txt`), 자동 생성된 노트
- `shasum -c` 결과가 `OK`
- 내려받은 바이너리의 `-version` 이 `v0.1.0`

## 자주 막히는 곳

- **`HTTP 403: Resource not accessible by integration`**: `permissions: contents: write` 가 빠졌습니다. 기본 토큰은 읽기 전용일 수 있습니다. **필요한 권한만 명시**하는 습관이 금융권 컴플라이언스 관점에서도 좋은 이야깃거리입니다.
- **태그를 push했는데 아무 일도 안 일어나요**: 태그가 `release.yml` 이 없는 옛 커밋을 가리키는지 확인하세요(`git log --oneline -1 v0.1.0`). 그리고 패턴이 `v*` 라서 `0.1.0` 처럼 `v` 없는 태그는 무시됩니다.
- **같은 태그로 다시 하고 싶어요**: Release와 태그를 지우고 다시 push 합니다.

  ```bash
  gh release delete v0.1.0 --cleanup-tag -y
  git tag -d v0.1.0
  ```

- **`-version` 이 여전히 `dev`**: `-X` 경로가 틀렸거나(`main.Version` ≠ `main.version`), `version` 을 `const` 로 선언한 경우입니다. 이때 Go는 **에러 없이 조용히 무시**합니다.
- **브라우저로 받은 바이너리가 맥에서 안 열려요**: macOS가 서명 안 된 파일을 막는 겁니다(Gatekeeper). `gh release download` 로 받으면 이 표시가 붙지 않습니다.

## 더 해보기 (선택)

- 크로스 컴파일을 `strategy: matrix` 로 바꿔 4개 잡이 **병렬로** 돌게 해 보세요. 이때는 각 잡이 `actions/upload-artifact` 로 올리고, 마지막 잡이 `actions/download-artifact` 로 모아서 Release를 만들어야 합니다. 실습 2의 "artifacts로 잡 사이에 넘긴다"와 같은 개념입니다.
- GitLab에도 같은 걸 만들 수 있습니다: `rules: - if: $CI_COMMIT_TAG` 인 잡에 `release:` 키워드(`tag_name: $CI_COMMIT_TAG`)를 쓰면 GitLab Releases가 생깁니다.

<details>
<summary>정답 예시 (먼저 직접 해 보고 여세요)</summary>

`main.go` — `func main()` 위:

```go
// version is overwritten at build time: go build -ldflags "-X main.version=v0.1.0"
var version = "dev"
```

`main.go` — `run()` 안:

```go
	apiURL := fs.String("api", "https://api.github.com", "GitHub API base URL")
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(out, version)
		return nil
	}
	if *repo == "" {
```

`.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags: ['v*']   # v0.1.0 같은 태그를 push 할 때만 실행

permissions:
  contents: write  # Release를 만들고 파일을 올리려면 쓰기 권한이 필요하다

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: false
      - name: Test
        run: go test ./...
      - name: Cross-compile
        run: |
          mkdir -p dist
          for target in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
            os=${target%/*}
            arch=${target#*/}
            ext=""
            if [ "$os" = windows ]; then ext=".exe"; fi
            CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
              -ldflags "-s -w -X main.version=${GITHUB_REF_NAME}" \
              -o "dist/actions-timer-${os}-${arch}${ext}" .
          done
          cd dist && sha256sum * > checksums.txt
      - name: Create GitHub Release
        env:
          GH_TOKEN: ${{ github.token }}
        run: gh release create "$GITHUB_REF_NAME" dist/* --title "$GITHUB_REF_NAME" --generate-notes
```

</details>
