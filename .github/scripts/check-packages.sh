#!/usr/bin/env bash
# Checks that the goreleaser output in dist/ (or $1) ships the binary and shell completions.
# Needs dpkg-deb and rpm.
set -euo pipefail
cd "${1:-dist}"

bash=usr/share/bash-completion/completions/env-exec
fish=usr/share/fish/vendor_completions.d/env-exec.fish
archived=(completions/env-exec.bash completions/_env-exec completions/env-exec.fish)

status=0
expect() {
    local artifact=$1 listing=$2 path
    shift 2
    for path in "$@"; do
        if ! grep -qxF -- "$path" <<<"$listing"; then
            echo "$artifact: missing $path"
            status=1
        fi
    done
}

for f in *.deb; do
    expect "$f" "$(dpkg-deb -c "$f" | awk '{print $6}' | sed 's|^\./||')" \
        usr/bin/env-exec "$bash" usr/share/zsh/vendor-completions/_env-exec "$fish"
done
for f in *.rpm; do
    expect "$f" "$(rpm -qlp "$f" | sed 's|^/||')" \
        usr/bin/env-exec "$bash" usr/share/zsh/site-functions/_env-exec "$fish"
done
for f in *.tar.gz; do
    expect "$f" "$(tar -tzf "$f")" env-exec README.md LICENSE "${archived[@]}"
done

pkgbuild=aur/env-exec-bin.pkgbuild
expect "$pkgbuild" "$(sed -n 's|.*"${pkgdir}/\([^"]*\)".*|\1|p' "$pkgbuild")" \
    usr/bin/env-exec "$bash" usr/share/zsh/site-functions/_env-exec "$fish"

cask=homebrew/Casks/env-exec.rb
expect "$cask" "$(sed -nE 's/^ *(bash|zsh|fish)_completion "(.*)"$/\2/p' "$cask")" "${archived[@]}"

if ((status == 0)); then
    set -- *.deb *.rpm *.tar.gz
    echo "ok: $# packages, $pkgbuild and $cask ship the completions"
fi
exit $status
