package main

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestRunMain_VersionFlag(t *testing.T) {
	version = "v0.0.0-test"

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runMain(context.Background(), []string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
	if out.String() != "v0.0.0-test\n" {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunMain_Help(t *testing.T) {
	version = "v0.0.0-test"

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runMain(context.Background(), []string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" {
		t.Fatalf("empty help output")
	}
}

func TestMain_UsesExitHook(t *testing.T) {
	version = "v0.0.0-test"

	oldExit := exit
	t.Cleanup(func() { exit = oldExit })

	var got int
	exit = func(code int) { got = code }

	os.Args = []string{"blu", "--version"}
	main()
	if got != 0 {
		t.Fatalf("exit=%d; want 0", got)
	}
}
