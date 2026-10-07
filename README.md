# Git Stack

Git Stack is a Go CLI for stacked branches. It reads `.git-stack`, shows branch state graphs, and runs cascade rebases from the current branch.

## Requirements

- Go 1.26 or newer
- Git
- Optional: Docker

## Install

Download the release archive for your OS and architecture from GitHub Releases, then install the binary.

### macOS

```bash
VERSION=v0.1.0-alpha.0
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=x64 ;;
  arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $ARCH"; exit 1 ;;
esac
curl -L -o /tmp/git-stack.tar.gz \
  "https://github.com/hermes-do-bruno/GitStack/releases/download/${VERSION}/git-stack-macos-${ARCH}.tar.gz"
tar -xzf /tmp/git-stack.tar.gz -C /tmp
sudo install -m 755 /tmp/git-stack /usr/local/bin/git-stack
```

### Linux

```bash
VERSION=v0.1.0-alpha.0
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=x64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $ARCH"; exit 1 ;;
esac
curl -L -o /tmp/git-stack.tar.gz \
  "https://github.com/hermes-do-bruno/GitStack/releases/download/${VERSION}/git-stack-linux-${ARCH}.tar.gz"
tar -xzf /tmp/git-stack.tar.gz -C /tmp
sudo install -m 755 /tmp/git-stack /usr/local/bin/git-stack
```

To verify the install:

```bash
git-stack help
```

## Build without Docker

Run the test suite first:

```bash
go test ./...
```

Build the binary:

```bash
mkdir -p bin
go build -o bin/git-stack ./cmd/git-stack
```

Run it locally:

```bash
./bin/git-stack help
```

## Build with Docker

Build the Docker image that already includes Go:

```bash
docker build -t git-stack-dev .
```

Run the tests inside the container:

```bash
docker run --rm -v "$PWD:/workspace" -w /workspace git-stack-dev go test ./...
```

Build the binary inside the container:

```bash
mkdir -p bin
docker run --rm -v "$PWD:/workspace" -w /workspace git-stack-dev go build -o bin/git-stack ./cmd/git-stack
```

Run the binary from the host after the Docker build:

```bash
./bin/git-stack help
```

## Release Workflow

Tag a release to trigger the GitHub Actions workflow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow builds release archives for:

- Linux x64
- Linux arm64
- macOS x64
- macOS arm64

Each archive includes the `git-stack` binary and a SHA-256 checksum.


- `git-stack parent [name]` — set the current branch parent(s) in `.git-stack`
- `git-stack graph` — show the branch graph and sync state
- `git-stack cascade [--apply]` — plan or apply the cascade from the current branch
