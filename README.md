# Git Stack

Git Stack is a Go CLI for stacked branches. It reads `.git-stack`, shows branch state graphs, and runs cascade rebases from the current branch.

## Requirements

- Go 1.26 or newer
- Git
- Optional: Docker

## Install

Use the install script to detect the current OS and architecture automatically.

```bash
VERSION=v0.1.0-alpha.2 ./scripts/install.sh
```

Or pass the version as an argument:

```bash
./scripts/install.sh v0.1.0-alpha.2
```

The script supports:

- Linux x64
- Linux arm64
- macOS x64
- macOS arm64

To verify the install:

```bash
git-stack version
```

## Build without Docker

Run the test suite first:

```bash
make test
```

Build the binary:

```bash
make build
```

The build reads the current git tag automatically. If the checkout is not on a tag, it falls back to `dev`.

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
docker run --rm -v "$PWD:/workspace" -w /workspace git-stack-dev make test
```

Build the binary inside the container:

```bash
docker run --rm -v "$PWD:/workspace" -w /workspace git-stack-dev make build
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
- `git-stack version` — show the CLI version
