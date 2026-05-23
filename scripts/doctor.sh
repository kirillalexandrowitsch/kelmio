#!/bin/sh
set -eu

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf 'Missing required command: %s\n' "$1" >&2
		exit 1
	fi
}

require_command curl
require_command docker
require_command go
require_command node
require_command npm

printf 'curl: '
curl --version | sed -n '1p'

printf 'docker: '
docker --version

printf 'docker compose: '
docker compose version

printf 'go: '
go version

printf 'node: '
node --version

printf 'npm: '
npm --version

printf 'Local toolchain check passed\n'
