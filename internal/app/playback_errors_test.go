package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPlayback_UnexpectedArgs(t *testing.T) {
	t.Parallel()

	cfgPath := writeTestConfig(t, "http://127.0.0.1:11000")

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "play", "--url", "http://x", "extra"}, &out, &errOut)
		if code != 2 || !strings.Contains(errOut.String(), "unexpected args") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	}

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "pause", "extra"}, &out, &errOut)
		if code != 2 || !strings.Contains(errOut.String(), "pause: unexpected args") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	}
}
