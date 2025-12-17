package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_CommandErrorBranches(t *testing.T) {
	t.Parallel()

	cfgPath := writeTestConfig(t, "http://127.0.0.1:11000")

	cases := []struct {
		args       []string
		wantCode   int
		wantStderr string
		noDiscover bool
		withConfig bool
	}{
		{args: []string{"volume"}, wantCode: 2, wantStderr: "volume: missing subcommand", withConfig: true},
		{args: []string{"volume", "set"}, wantCode: 2, wantStderr: "volume set: missing level", withConfig: true},
		{args: []string{"volume", "nope"}, wantCode: 2, wantStderr: "volume: unknown subcommand", withConfig: true},
		{args: []string{"mute"}, wantCode: 2, wantStderr: "mute: missing subcommand", withConfig: true},
		{args: []string{"mute", "nope"}, wantCode: 2, wantStderr: "mute: unknown subcommand", withConfig: true},
		{args: []string{"group"}, wantCode: 2, wantStderr: "group: missing subcommand", withConfig: true},
		{args: []string{"group", "add"}, wantCode: 2, wantStderr: "group add: missing slave", withConfig: true},
		{args: []string{"group", "remove"}, wantCode: 2, wantStderr: "group remove: missing slave", withConfig: true},
		{args: []string{"group", "nope"}, wantCode: 2, wantStderr: "group: unknown subcommand", withConfig: true},
		{args: []string{"queue"}, wantCode: 2, wantStderr: "queue: missing subcommand", withConfig: true},
		{args: []string{"queue", "delete"}, wantCode: 2, wantStderr: "queue delete: missing id", withConfig: true},
		{args: []string{"queue", "move"}, wantCode: 2, wantStderr: "queue move: usage", withConfig: true},
		{args: []string{"queue", "save"}, wantCode: 2, wantStderr: "queue save: missing name", withConfig: true},
		{args: []string{"queue", "nope"}, wantCode: 2, wantStderr: "queue: unknown subcommand", withConfig: true},
		{args: []string{"presets"}, wantCode: 2, wantStderr: "presets: missing subcommand", withConfig: true},
		{args: []string{"presets", "load"}, wantCode: 2, wantStderr: "presets load: missing id", withConfig: true},
		{args: []string{"presets", "nope"}, wantCode: 2, wantStderr: "presets: unknown subcommand", withConfig: true},
		{args: []string{"tunein"}, wantCode: 2, wantStderr: "tunein: missing subcommand", withConfig: true},
		{args: []string{"tunein", "play"}, wantCode: 2, wantStderr: "tunein play: missing query", withConfig: true},
		{args: []string{"tunein", "search"}, wantCode: 2, wantStderr: "tunein search: missing query", withConfig: true},
	}

	for _, tc := range cases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer

			runArgs := make([]string, 0, 8)
			if tc.withConfig {
				runArgs = append(runArgs, "--config", cfgPath, "--discover=false")
			}
			runArgs = append(runArgs, tc.args...)

			code := Run(context.Background(), runArgs, &out, &errOut)
			if code != tc.wantCode {
				t.Fatalf("code=%d want=%d stderr=%q", code, tc.wantCode, errOut.String())
			}
			if tc.wantStderr != "" && !strings.Contains(errOut.String(), tc.wantStderr) {
				t.Fatalf("stderr=%q; want contains %q", errOut.String(), tc.wantStderr)
			}
		})
	}
}
