#!/usr/bin/env bash

set -euo pipefail

project_root=$(cd "$(dirname "$0")/.." && pwd)
manifest="$project_root/docs/results/evidence-manifest.json"

if [[ ${EVIDENCE_CHECKS_PASSED:-} != 1 ]]; then
  printf 'run this generator through make evidence so all recorded checks execute first\n' >&2
  exit 1
fi

cd "$project_root"

git_commit=$(git rev-parse HEAD)

source_sha=$(
  {
    find cmd internal scenarios scripts -type f \( -name '*.go' -o -name '*.json' -o -name '*.sh' \)
    printf '%s\n' Makefile go.mod Dockerfile docker-compose.yml .github/workflows/ci.yml
  } |
    LC_ALL=C sort |
    while IFS= read -r file; do shasum -a 256 "$file"; done |
    shasum -a 256 |
    awk '{print $1}'
)

generated_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
go_version=$(go version)
summary_csv_sha=$(shasum -a 256 docs/results/summary.csv | awk '{print $1}')
summary_json_sha=$(shasum -a 256 docs/results/summary.json | awk '{print $1}')

{
  printf '{\n'
  printf '  "schemaVersion": 1,\n'
  printf '  "generatedAt": "%s",\n' "$generated_at"
  printf '  "source": {\n'
  printf '    "baseGitCommit": "%s",\n' "$git_commit"
  printf '    "aggregateSha256": "%s"\n' "$source_sha"
  printf '  },\n'
  printf '  "toolchain": {"goVersion": "%s"},\n' "$go_version"
  printf '  "commands": [\n'
  printf '    "make fmt-check",\n'
  printf '    "make vet",\n'
  printf '    "make test",\n'
  printf '    "make race",\n'
  printf '    "make build",\n'
  printf '    "make evidence"\n'
  printf '  ],\n'
  printf '  "checks": {\n'
  printf '    "format": "passed",\n'
  printf '    "vet": "passed",\n'
  printf '    "tests": "passed",\n'
  printf '    "race": "passed",\n'
  printf '    "build": "passed"\n'
  printf '  },\n'
  printf '  "scenarioSha256": {\n'
  first=true
  for scenario in $(LC_ALL=C find scenarios -maxdepth 1 -type f -name '*.json' | sort); do
    if $first; then
      first=false
    else
      printf ',\n'
    fi
    printf '    "%s": "%s"' "$scenario" "$(shasum -a 256 "$scenario" | awk '{print $1}')"
  done
  printf '\n  },\n'
  printf '  "resultSha256": {\n'
  printf '    "docs/results/summary.csv": "%s",\n' "$summary_csv_sha"
  printf '    "docs/results/summary.json": "%s"\n' "$summary_json_sha"
  printf '  }\n'
  printf '}\n'
} > "$manifest"
