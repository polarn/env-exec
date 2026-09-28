#!/usr/bin/env bash
# Checks the bash, fish and zsh completions against a stub command.
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

mkdir "$tmp/zdot"
cat >"$tmp/zdot/.zshrc" <<'EOF'
PS1='<PROMPT>'
path=("${ZDOTDIR:h}/bin")
fpath=("$COMPLETIONS" $fpath)
autoload -Uz compinit && compinit -u -D
zmodload zsh/complist
LISTMAX=1000000
zstyle ':completion:*' verbose no
zstyle ':completion:*:default' list-colors 'no=NO' 'lc=<LC>' 'rc=<RC>' 'ec=<EC>'
_stubcmd() { compadd plan apply }
compdef _stubcmd stubcmd
_test_list() { zle list-choices }
_test_end() { print -r '<END>' }
zle -N _test_list
zle -N _test_end
bindkey '^E' _test_list
bindkey '^A' _test_end
EOF

cat >"$tmp/complete.zsh" <<'EOF'
zmodload zsh/zpty
zpty z "ZDOTDIR=${(q)1} zsh -d -i"
zpty -r -m z out '*<PROMPT>*' || exit 1
zpty -w -n z "$2"$'\C-E\C-A'
out=
while zpty -r z line; do
    out+=$line
    [[ $line == *'<END>'* ]] && break
done
zpty -d z
[[ $out == *'<END>'* ]] || exit 1
print -r -- "$out"
EOF

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

complete_zsh() {
    local out
    out=$(COMPLETIONS=$dir timeout 30 zsh -f "$tmp/complete.zsh" "$tmp/zdot" "$1") || {
        echo 'zsh driver failed'
        return
    }
    grep -o '<LC>NO<RC>[^<]*<EC>' <<<"$out" | sed 's/^<LC>NO<RC>//; s/<EC>$//'
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

declare -A zsh_want=(
    ['env-exec -']='- --dry-run --help --version -h -n -v'
)

for ((i = 0; i < ${#cases[@]}; i += 2)); do
    line=${cases[i]}
    for shell in bash fish zsh; do
        want=${cases[i + 1]}
        if [[ $shell == zsh && ${zsh_want[$line]+set} ]]; then
            want=${zsh_want[$line]}
        fi
        check "$shell" "$line" "$want"
    done
done

exit $status
