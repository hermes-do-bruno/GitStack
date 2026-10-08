#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install.sh [VERSION]

Install the Git Stack release for the current OS and architecture.

Environment variables:
  VERSION   Release tag to install, for example v0.1.0-alpha.2
  PREFIX    Install prefix (default: /usr/local)

Examples:
  VERSION=v0.1.0-alpha.2 ./scripts/install.sh
  ./scripts/install.sh v0.1.0-alpha.2
EOF
}

if [[ ${1:-} == "-h" || ${1:-} == "--help" ]]; then
  usage
  exit 0
fi

version="${VERSION:-${1:-}}"
if [[ -z "$version" ]]; then
  usage >&2
  exit 1
fi

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) platform="macos" ;;
  Linux) platform="linux" ;;
  *)
    echo "unsupported operating system: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="x64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="git-stack-${platform}-${arch}.tar.gz"
url="https://github.com/hermes-do-bruno/GitStack/releases/download/${version}/${asset}"

if [[ ${PRINT_URL:-0} == 1 ]]; then
  printf '%s\n' "$url"
  exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

curl -fsSL -o "$tmpdir/$asset" "$url"
tar -xzf "$tmpdir/$asset" -C "$tmpdir"

prefix="${PREFIX:-/usr/local}"
install_dir="$prefix/bin"
mkdir -p "$install_dir"
install -m 755 "$tmpdir/git-stack" "$install_dir/git-stack"

echo "installed git-stack to $install_dir/git-stack"
