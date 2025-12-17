package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/steipete/blucli/internal/bluos"
)

func TestRunNow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status state="play" volume="15" mute="0" artist="A" title1="T"/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "now"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !strings.Contains(got, "A — T") {
		t.Fatalf("stdout = %q; want now output", got)
	}
}

func TestFlattenAudio(t *testing.T) {
	t.Parallel()

	got := flattenAudio(bluos.RadioBrowse{
		Categories: []bluos.RadioCategory{{
			Text: "Stations",
			Items: []bluos.RadioItem{
				{ID: "x", Type: " audio ", URL: "TuneIn%3Ax"},
				{ID: "y", Type: "link"},
			},
		}},
		Items: []bluos.RadioItem{
			{ID: "z", Type: "AuDiO", URL: "TuneIn%3Az"},
		},
	})
	if len(got) != 2 || got[0].ID != "x" || got[1].ID != "z" {
		t.Fatalf("audio = %+v; want ids x,z", got)
	}
}
