package app

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func TestSpotifyLogin_SuccessRoundTrip(t *testing.T) {
	var gotForm string
	accountsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotForm = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","refresh_token":"RT","token_type":"Bearer","scope":"s","expires_in":3600}`))
	}))
	t.Cleanup(accountsSrv.Close)
	t.Setenv("BLU_SPOTIFY_ACCOUNTS_BASE_URL", accountsSrv.URL)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	redirect := "http://" + addr + "/callback"

	paths := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}
	cfg := config.Config{}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	t.Cleanup(func() { _ = stdoutR.Close(); _ = stdoutW.Close(); _ = stderrR.Close(); _ = stderrW.Close() })

	out := output.New(output.Options{Stdout: stdoutW, Stderr: stderrW})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan int, 1)
	go func() {
		done <- cmdSpotifyLogin(ctx, out, paths, cfg, []string{"--client-id", "CID", "--redirect", redirect, "--no-open"})
		_ = stdoutW.Close()
		_ = stderrW.Close()
	}()

	br := bufio.NewReader(stdoutR)
	authURLLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read auth url: %v", err)
	}
	authURLLine = strings.TrimSpace(authURLLine)
	u, err := url.Parse(authURLLine)
	if err != nil {
		t.Fatalf("parse auth url: %v (%q)", err, authURLLine)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatalf("missing state in %q", authURLLine)
	}

	cbURL, _ := url.Parse(redirect)
	q := cbURL.Query()
	q.Set("state", state)
	q.Set("code", "CODE")
	cbURL.RawQuery = q.Encode()

	resp, err := http.Get(cbURL.String())
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status=%d", resp.StatusCode)
	}

	code := <-done
	if code != 0 {
		var errBuf bytes.Buffer
		_, _ = errBuf.ReadFrom(stderrR)
		t.Fatalf("code=%d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(gotForm, "grant_type=authorization_code") || !strings.Contains(gotForm, "code=CODE") {
		t.Fatalf("token form = %q", gotForm)
	}

	data, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !bytes.Contains(data, []byte(`"access_token": "AT"`)) || !bytes.Contains(data, []byte(`"refresh_token": "RT"`)) {
		t.Fatalf("config = %s", string(data))
	}
}
