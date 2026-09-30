#!/bin/sh
set -eu

fail() {
  printf 'Erro: %s\n' "$*" >&2
  exit 1
}

case "$(uname -s)/$(uname -m)" in
  Linux/x86_64|Linux/amd64) asset=notion-getpage_linux_amd64 ;;
  *) fail 'esta versão do instalador suporta apenas Linux x86-64' ;;
esac

command -v sha256sum >/dev/null 2>&1 || fail 'sha256sum é necessário para verificar o download'
if command -v curl >/dev/null 2>&1; then
  download() { curl --fail --location --silent --show-error --retry 3 --output "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  download() { wget --quiet --output-document="$2" "$1"; }
else
  fail 'curl ou wget é necessário para baixar a release'
fi

repository=${NOTION_GETPAGE_REPOSITORY:-alepspizzetti/notion-get-page}
version=${NOTION_GETPAGE_VERSION:-latest}
case "$repository" in
  */*) ;;
  *) fail 'repositório de releases ainda não configurado' ;;
esac
case "$version" in
  latest) release_path=latest/download ;;
  v[0-9]*)
    case "$version" in *[!a-zA-Z0-9._-]*) fail 'versão inválida' ;; esac
    release_path="download/$version"
    ;;
  *) fail 'use NOTION_GETPAGE_VERSION=latest ou uma tag como v1.0.0' ;;
esac

base_url=${NOTION_GETPAGE_RELEASE_BASE_URL:-https://github.com/$repository/releases/$release_path}
install_dir=${NOTION_GETPAGE_INSTALL_DIR:-"$HOME/.local/bin"}
doc_dir=${NOTION_GETPAGE_DOC_DIR:-"$HOME/.local/share/doc/notion-getpage"}
tmp_dir=$(mktemp -d) || fail 'não foi possível criar diretório temporário'
tmp_installed=
cleanup() {
  if [ -n "$tmp_installed" ]; then rm -f "$tmp_installed"; fi
  rm -rf "$tmp_dir"
}
trap cleanup 0
trap 'exit 1' 1 2 3 15

download "$base_url/$asset" "$tmp_dir/$asset" || fail 'falha ao baixar o binário'
download "$base_url/checksums.txt" "$tmp_dir/checksums.txt" || fail 'falha ao baixar os checksums'
download "$base_url/LICENSE" "$tmp_dir/LICENSE" || fail 'falha ao baixar a licença'
download "$base_url/THIRD_PARTY_NOTICES.txt" "$tmp_dir/THIRD_PARTY_NOTICES.txt" || fail 'falha ao baixar os avisos de terceiros'
for file in "$asset" LICENSE THIRD_PARTY_NOTICES.txt; do
  expected=$(awk -v name="$file" '$2 == name { print $1 }' "$tmp_dir/checksums.txt")
  case "$expected" in *[!0-9a-fA-F]*) fail 'checksum inválido na release' ;; esac
  [ "${#expected}" -eq 64 ] || fail 'checksum ausente ou inválido na release'
  printf '%s  %s\n' "$expected" "$file" | (cd "$tmp_dir" && sha256sum --check --status -) || fail "checksum não confere: $file"
done

mkdir -p "$install_dir" || fail 'não foi possível criar o diretório de instalação'
mkdir -p "$doc_dir" || fail 'não foi possível criar o diretório de licenças'
[ ! -d "$install_dir/notion-getpage" ] || fail 'o destino é um diretório'
tmp_installed=$(mktemp "$install_dir/.notion-getpage.XXXXXX") || fail 'não foi possível preparar a instalação'
cp "$tmp_dir/$asset" "$tmp_installed" || fail 'não foi possível copiar o binário'
chmod 0755 "$tmp_installed" || fail 'não foi possível tornar o binário executável'
cp "$tmp_dir/LICENSE" "$doc_dir/LICENSE" || fail 'não foi possível instalar a licença'
cp "$tmp_dir/THIRD_PARTY_NOTICES.txt" "$doc_dir/THIRD_PARTY_NOTICES.txt" || fail 'não foi possível instalar os avisos de terceiros'
mv -f "$tmp_installed" "$install_dir/notion-getpage" || fail 'não foi possível concluir a instalação'
tmp_installed=

printf 'Instalado em %s/notion-getpage\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Adicione %s ao PATH para usar o comando sem o caminho completo.\n' "$install_dir" ;;
esac
printf 'Depois, execute: %s/notion-getpage login\n' "$install_dir"
