package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunVolumeGetUpDown(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/Status":
			_, _ = w.Write([]byte(`<status state="play" volume="15" mute="0" db="-40.0"/>`))
		default:
			_, _ = w.Write([]byte(`<ok/>`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "volume", "get"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("volume get code=%d stderr=%q", code, errOut.String())
		}
		if got := out.String(); !strings.Contains(got, "\"volume\": 15") || !strings.Contains(got, "\"db\": -40") {
			t.Fatalf("stdout = %q", got)
		}
	}

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "volume", "up"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("volume up code=%d stderr=%q", code, errOut.String())
		}
	}

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "volume", "down"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("volume down code=%d stderr=%q", code, errOut.String())
		}
	}

	want := []string{"/Status", "/Volume?db=2&tell_slaves=1", "/Volume?db=-2&tell_slaves=1"}
	for i := range want {
		if got := <-seen; got != want[i] {
			t.Fatalf("call[%d]=%q want %q", i, got, want[i])
		}
	}
}

func TestRunMuteOnOff(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	for _, cmd := range [][]string{{"mute", "on"}, {"mute", "off"}} {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), append([]string{"--config", cfgPath, "--discover=false"}, cmd...), &out, &errOut)
		if code != 0 {
			t.Fatalf("%v code=%d stderr=%q", cmd, code, errOut.String())
		}
	}

	want := []string{"/Volume?mute=1&tell_slaves=1", "/Volume?mute=0&tell_slaves=1"}
	for i := range want {
		if got := <-seen; got != want[i] {
			t.Fatalf("call[%d]=%q want %q", i, got, want[i])
		}
	}
}

func TestRunGroupStatusAndRemove(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/SyncStatus":
			_, _ = w.Write([]byte(`<SyncStatus group="G" volume="10" mute="0"><slave id="s1" port="11001"/></SyncStatus>`))
		default:
			_, _ = w.Write([]byte(`<ok/>`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "group", "status"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("group status code=%d stderr=%q", code, errOut.String())
		}
		if got := out.String(); !strings.Contains(got, "group: G") {
			t.Fatalf("stdout=%q", got)
		}
	}

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "group", "remove", "192.0.2.2:11001"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("group remove code=%d stderr=%q", code, errOut.String())
		}
	}

	want := []string{"/SyncStatus", "/RemoveSlave?port=11001&slave=192.0.2.2"}
	for i := range want {
		if got := <-seen; got != want[i] {
			t.Fatalf("call[%d]=%q want %q", i, got, want[i])
		}
	}
}

func TestRunTuneInPlayByIDAndQuery(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/RadioBrowse":
			_, _ = w.Write([]byte(`<radiotime service="TuneIn"><category text="Stations"><item id="s1" text="X" type="audio" URL="TuneIn%3As1"/></category></radiotime>`))
		default:
			_, _ = w.Write([]byte(`<ok/>`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "tunein", "play", "--id", "s21750"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("tunein play --id code=%d stderr=%q", code, errOut.String())
		}
	}

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "tunein", "play", "Garrett"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("tunein play query code=%d stderr=%q", code, errOut.String())
		}
	}

	want := []string{
		"/Play?url=TuneIn%3As21750",
		"/RadioBrowse?expr=Garrett&service=TuneIn",
		"/Play?url=TuneIn%3As1",
	}
	for i := range want {
		if got := <-seen; got != want[i] {
			t.Fatalf("call[%d]=%q want %q", i, got, want[i])
		}
	}
}

func TestShuffleRepeatUsageErrors(t *testing.T) {
	t.Parallel()

	cfgPath := writeTestConfig(t, "http://127.0.0.1:11000")

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "shuffle"}, &out, &errOut)
		if code != 2 || !strings.Contains(errOut.String(), "missing arg") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	}
	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "repeat", "nope"}, &out, &errOut)
		if code != 2 || !strings.Contains(errOut.String(), "unknown arg") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	}
}

func TestUsageCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if !usageCommand(&out, "spotify") {
		t.Fatalf("want true")
	}
	if got := out.String(); !strings.Contains(got, "blu spotify login") {
		t.Fatalf("out=%q", got)
	}
}
