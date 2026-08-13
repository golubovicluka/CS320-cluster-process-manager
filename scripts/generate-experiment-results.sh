#!/usr/bin/env bash

set -euo pipefail

project_root=$(cd "$(dirname "$0")/.." && pwd)
results_dir="$project_root/docs/results"
temporary_dir=$(mktemp -d)
trap 'rm -rf "$temporary_dir"' EXIT

runs=(
  'scenarios/balanced.json|'
  'scenarios/balanced.json|least-loaded'
  'scenarios/heterogeneous.json|round-robin'
  'scenarios/heterogeneous.json|least-loaded'
  'scenarios/overload.json|'
  'scenarios/priority-workload.json|'
  'scenarios/node-failure.json|'
)

mkdir -p "$results_dir"
: > "$results_dir/summary.csv"
printf '[\n' > "$results_dir/summary.json"

first_run=true
run_number=0
for run in "${runs[@]}"; do
  IFS='|' read -r scenario scheduler <<< "$run"
  run_number=$((run_number + 1))
  json_output="$temporary_dir/run-$run_number.json"
  csv_output="$temporary_dir/run-$run_number.csv"
  command=(go run ./cmd/simulator -scenario "$scenario")
  if [[ -n "$scheduler" ]]; then
    command+=(-scheduler "$scheduler")
  fi

  (
    cd "$project_root"
    "${command[@]}" -format json -output "$json_output"
    "${command[@]}" -format csv -output "$csv_output"
  )

  if $first_run; then
    cat "$csv_output" > "$results_dir/summary.csv"
  else
    tail -n 1 "$csv_output" >> "$results_dir/summary.csv"
    printf ',\n' >> "$results_dir/summary.json"
  fi
  sed 's/^/  /' "$json_output" >> "$results_dir/summary.json"
  first_run=false
done

printf ']\n' >> "$results_dir/summary.json"
