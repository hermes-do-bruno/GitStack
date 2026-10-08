#!/bin/sh
set -eu

repo="hermes-do-bruno/GitStack"

usage() {
  cat <<'EOF'
Usage: install.sh [VERSION]

Install the Git Stack release for the current OS and architecture.
If VERSION is omitted, the script fetches the latest GitHub release tag.

Environment variables:
  VERSION   Release tag to install, for example v0.1.0-alpha.2
            Use "latest" to fetch the newest GitHub release tag.
  PREFIX    Install prefix (default: /usr/local)

Examples:
  ./scripts/install.sh
  VERSION=v0.1.0-alpha.2 ./scripts/install.sh
  ./scripts/install.sh v0.1.0-alpha.2
EOF
}

case "${1-}" in
  -h|--help)
    usage
    exit 0
    ;;
esac

latest_version() {
  curl -fsSL \
    -H 'Accept: application/vnd.github+json' \
    -H 'User-Agent: GitStack installer' \
    "https://api.github.com/repos/${repo}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n1
}

version="${VERSION-}"
if [ -z "$version" ] && [ -n "${1-}" ]; then
  version="$1"
fi
if [ -z "$version" ] || [ "$version" = "latest" ]; then
  version="$(latest_version)"
fi

if [ -z "$version" ]; then
  echo "could not determine the latest GitHub release tag" >&2
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
url="https://github.com/${repo}/releases/download/${version}/${asset}"

if [ "${PRINT_URL-0}" = 1 ]; then
  printf '%s\n' "$url"
  exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

curl -fsSL -o "$tmpdir/$asset" "$url"
tar -xzf "$tmpdir/$asset" -C "$tmpdir"

prefix="${PREFIX-/usr/local}"
install_dir="$prefix/bin"
mkdir -p "$install_dir"
install -m 755 "$tmpdir/git-stack" "$install_dir/git-stack"

echo "installed git-stack to $install_dir/git-stack"
