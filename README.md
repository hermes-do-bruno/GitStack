# Git Stack

Git Stack is a Go CLI for stacked branches. It reads `.git-cascade`, shows branch state graphs, and runs cascade rebases from the current branch.

## Requirements

- Go 1.26 or newer
- Git
- Optional: Docker

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

## Commands

- `git-stack parent [name]` — set the current branch parent(s) in `.git-cascade`
- `git-stack graph` — show the branch graph and sync state
- `git-stack cascade [--apply]` — plan or apply the cascade from the current branch
