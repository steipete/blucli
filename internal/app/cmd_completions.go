package app

import (
	"fmt"

	"github.com/steipete/blucli/internal/output"
)

func cmdCompletions(out *output.Printer, args []string) int {
	if len(args) == 0 {
		out.Errorf("completions: missing shell (bash|zsh)")
		return 2
	}

	switch args[0] {
	case "bash":
		fmt.Fprint(out.Stdout(), bashCompletionScript())
		return 0
	case "zsh":
		fmt.Fprint(out.Stdout(), zshCompletionScript())
		return 0
	default:
		out.Errorf("completions: unknown shell %q (expected bash|zsh)", args[0])
		return 2
	}
}

func bashCompletionScript() string {
	return `# bash completion for blu
_blu_complete() {
  local cur cmd
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"

  # global flags (best-effort)
  if [[ "$cur" == -* ]]; then
    COMPREPLY=( $(compgen -W "--device --json --timeout --dry-run --trace-http --version -v --discover --discover-timeout --config -h --help" -- "$cur") )
    return 0
  fi

  cmd="${COMP_WORDS[1]}"
  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "version completions devices status now watch play pause stop next prev shuffle repeat volume mute group queue presets browse playlists inputs sleep diag doctor raw help" -- "$cur") )
    return 0
  fi

  case "$cmd" in
    completions)
      COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
      ;;
    watch)
      COMPREPLY=( $(compgen -W "status sync" -- "$cur") )
      ;;
    play)
      if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "--url --seek --id" -- "$cur") )
      fi
      ;;
    shuffle)
      COMPREPLY=( $(compgen -W "on off" -- "$cur") )
      ;;
    repeat)
      COMPREPLY=( $(compgen -W "off track queue" -- "$cur") )
      ;;
    volume)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "get set up down" -- "$cur") )
      fi
      ;;
    mute)
      COMPREPLY=( $(compgen -W "on off toggle" -- "$cur") )
      ;;
    group)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "status add remove" -- "$cur") )
      else
        if [[ "$cur" == -* ]]; then
          COMPREPLY=( $(compgen -W "--name" -- "$cur") )
        fi
      fi
      ;;
    queue)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "list clear delete move save" -- "$cur") )
      else
        if [[ "$cur" == -* ]]; then
          COMPREPLY=( $(compgen -W "--start --end" -- "$cur") )
        fi
      fi
      ;;
    presets)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "list load" -- "$cur") )
      fi
      ;;
    browse)
      if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "--key --q --context" -- "$cur") )
      fi
      ;;
    playlists)
      if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "--service --category --expr" -- "$cur") )
      fi
      ;;
    inputs)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "list play" -- "$cur") )
      fi
      ;;
    raw)
      if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "--param --write" -- "$cur") )
      fi
      ;;
  esac
  return 0
}

complete -F _blu_complete blu
`
}

func zshCompletionScript() string {
	return `#compdef blu

autoload -U +X bashcompinit && bashcompinit
source <(blu completions bash)
`
}
