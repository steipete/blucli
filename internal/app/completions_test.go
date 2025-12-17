package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunCompletionsBash(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"completions", "bash"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "# bash completion for blu") || !strings.Contains(got, "complete -F _blu_complete blu") {
		t.Fatalf("stdout = %q; want bash completion script", got)
	}
}

func TestRunCompletionsZsh(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"completions", "zsh"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "#compdef blu") || !strings.Contains(got, "blu completions bash") {
		t.Fatalf("stdout = %q; want zsh completion script", got)
	}
}
