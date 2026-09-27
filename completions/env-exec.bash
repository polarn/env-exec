_env_exec() {
    local cur prev words cword comp_args
    if declare -F _comp_initialize >/dev/null; then
        _comp_initialize -- "$@" || return
    else
        _init_completion || return
    fi

    local i
    for ((i = 1; i < cword; i++)); do
        case ${words[i]} in
            -n | --dry-run) ;;
            -h | --help | -v | --version) return ;;
            *) break ;;
        esac
    done

    if ((i == cword)) && [[ $cur == -* ]]; then
        mapfile -t COMPREPLY < <(compgen -W '-h --help -v --version -n --dry-run' -- "$cur")
        return
    fi

    if declare -F _comp_command_offset >/dev/null; then
        _comp_command_offset "$i"
    else
        _command_offset "$i"
    fi
}

complete -F _env_exec env-exec
