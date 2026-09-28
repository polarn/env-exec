function __env_exec_no_command
    for token in (commandline -opc)[2..]
        contains -- $token -n --dry-run
        or return 1
    end
end

function __env_exec_complete
    for token in (commandline -opc)[2..]
        switch $token
            case -n --dry-run
            case -h --help -v --version
                return
            case '*'
                break
        end
    end
    __fish_complete_subcommand
end

complete -c env-exec -f
complete -c env-exec -n __env_exec_no_command -s h -l help -d 'Show this help message'
complete -c env-exec -n __env_exec_no_command -s v -l version -d 'Show version information'
complete -c env-exec -n __env_exec_no_command -s n -l dry-run -d 'Print environment variables without executing command'
complete -c env-exec -a '(__env_exec_complete)'
