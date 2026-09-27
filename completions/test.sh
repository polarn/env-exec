#!/usr/bin/env bash
# Checks the bash and fish completions against a stub command and parses the zsh one.
# Needs bash-completion 2.x, fish and zsh. BASH_COMPLETION overrides the bash_completion path.
set -euo pipefail
export LC_ALL=C

dir=$(cd "$(dirname "$0")" && pwd)
bash_completion=${BASH_COMPLETION:-/usr/share/bash-completion/bash_completion}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/bin"
printf '#!/bin/sh\n' >"$tmp/bin/stubcmd"
chmod +x "$tmp/bin/stubcmd"
export PATH="$tmp/bin:$PATH"
cd "$tmp"

complete_bash() (
    set +euo pipefail
    source "$bash_completion"
    source "$dir/env-exec.bash"
    compopt() { :; }
    complete -W 'plan apply' stubcmd
    read -ra COMP_WORDS <<<"$1"
    [[ $1 == *' ' ]] && COMP_WORDS+=('')
    COMP_CWORD=$((${#COMP_WORDS[@]} - 1))
    COMP_LINE=$1
    COMP_POINT=${#1}
    spec=$(complete -p env-exec)
    [[ $spec =~ -F\ ([^ ]+) ]] || exit 1
    "${BASH_REMATCH[1]}" env-exec "${COMP_WORDS[COMP_CWORD]}" "${COMP_WORDS[COMP_CWORD - 1]}"
    printf '%s\n' "${COMPREPLY[@]}"
)

complete_fish() {
    fish --no-config -c '
        source $argv[1]
        complete -c stubcmd -f -a "plan apply"
        complete -C $argv[2]
    ' "$dir/env-exec.fish" "$1"
}

status=0
check() {
    local shell=$1 line=$2 want=$3 got
    got=$("complete_$shell" "$line" | cut -f1 | sed '/^$/d' | sort | paste -sd ' ' -) || true
    if [[ $got == "$want" ]]; then
        printf 'ok    %-4s %s\n' "$shell" "$line|"
    else
        printf 'FAIL  %-4s %s\n      want: %s\n      got:  %s\n' "$shell" "$line|" "$want" "$got"
        status=1
    fi
}

cases=(
    'env-exec -' '--dry-run --help --version -h -n -v'
    'env-exec --d' '--dry-run'
    'env-exec stubc' 'stubcmd'
    'env-exec -n stubc' 'stubcmd'
    'env-exec stubcmd ' 'apply plan'
    'env-exec stubcmd p' 'plan'
    'env-exec --dry-run stubcmd a' 'apply'
    'env-exec stubcmd -n ' 'apply plan'
    'env-exec stubcmd -' ''
    'env-exec --version ' ''
    'env-exec -h ' ''
)

for ((i = 0; i < ${#cases[@]}; i += 2)); do
    for shell in bash fish; do
        check "$shell" "${cases[i]}" "${cases[i + 1]}"
    done
done

if zsh -n "$dir/_env-exec"; then
    echo 'ok    zsh  _env-exec parses'
else
    status=1
fi

exit $status
