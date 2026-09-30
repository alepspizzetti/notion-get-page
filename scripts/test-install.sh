#!/bin/sh
set -eu

[ "$#" -eq 1 ] || { printf 'Uso: %s BINARIO\n' "$0" >&2; exit 2; }
binary=$1
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' 0
trap 'exit 1' 1 2 3 15
mkdir -p "$tmp_dir/release" "$tmp_dir/install"
cp "$binary" "$tmp_dir/release/notion-getpage_linux_amd64"
cp LICENSE "$tmp_dir/release/LICENSE"
printf 'Third-party notices for installer test\n' > "$tmp_dir/release/THIRD_PARTY_NOTICES.txt"
(cd "$tmp_dir/release" && sha256sum notion-getpage_linux_amd64 LICENSE THIRD_PARTY_NOTICES.txt > checksums.txt)

NOTION_GETPAGE_REPOSITORY=test/notion-getpage \
NOTION_GETPAGE_RELEASE_BASE_URL="file://$tmp_dir/release" \
NOTION_GETPAGE_INSTALL_DIR="$tmp_dir/install" \
NOTION_GETPAGE_DOC_DIR="$tmp_dir/docs" \
sh install.sh
cmp "$binary" "$tmp_dir/install/notion-getpage"
cmp LICENSE "$tmp_dir/docs/LICENSE"
cmp "$tmp_dir/release/THIRD_PARTY_NOTICES.txt" "$tmp_dir/docs/THIRD_PARTY_NOTICES.txt"

printf 'corrupted' > "$tmp_dir/release/notion-getpage_linux_amd64"
if NOTION_GETPAGE_REPOSITORY=test/notion-getpage \
   NOTION_GETPAGE_RELEASE_BASE_URL="file://$tmp_dir/release" \
   NOTION_GETPAGE_INSTALL_DIR="$tmp_dir/install" \
   NOTION_GETPAGE_DOC_DIR="$tmp_dir/docs" \
   sh install.sh >/dev/null 2>&1; then
  printf 'Erro: instalador aceitou binário com checksum inválido\n' >&2
  exit 1
fi
cmp "$binary" "$tmp_dir/install/notion-getpage"
printf 'Instalador validado.\n'
