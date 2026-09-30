#!/bin/sh
set -eu

[ "$#" -eq 1 ] || { printf 'Uso: %s ARQUIVO\n' "$0" >&2; exit 2; }
output=$1
module_list=$(mktemp)
trap 'rm -f "$module_list"' 0
trap 'exit 1' 1 2 3 15
go list -m -f '{{if not .Main}}{{.Path}} {{.Version}} {{.Dir}}{{end}}' all > "$module_list"

{
  printf 'Third-party license notices for notion-getpage\n\n'
  while read -r module version directory; do
    [ -n "$module" ] || continue
    [ -f "$directory/LICENSE" ] || { printf 'Missing license: %s\n' "$module" >&2; exit 1; }
    printf '%s %s\n' "$module" "$version"
    printf '%s\n' '------------------------------------------------------------'
    cat "$directory/LICENSE"
    printf '\n\n'
  done < "$module_list"
} > "$output"
