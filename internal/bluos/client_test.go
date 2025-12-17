package bluos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestStatusParsingMuteInt(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			t.Fatalf("path = %q; want /Status", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status state="play" volume="15" mute="1" db="-40.0" artist="Artist" title1="Title" etag="12"/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{Timeout: 2 * time.Second})

	status, err := client.Status(context.Background(), StatusOptions{})
	if err != nil {
		t.Fatalf("Status() err = %v", err)
	}
	if !bool(status.Mute) {
		t.Fatalf("Mute = %v; want true", status.Mute)
	}
	if status.Volume != 15 {
		t.Fatalf("Volume = %d; want 15", status.Volume)
	}
}

func TestStatusParsingElements(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			t.Fatalf("path = %q; want /Status", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<status etag="123"><db>-58.5</db><mute>0</mute><state>stop</state><volume>5</volume></status>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{Timeout: 2 * time.Second})

	status, err := client.Status(context.Background(), StatusOptions{})
	if err != nil {
		t.Fatalf("Status() err = %v", err)
	}
	if status.State != "stop" {
		t.Fatalf("State = %q; want stop", status.State)
	}
	if status.Volume != 5 {
		t.Fatalf("Volume = %d; want 5", status.Volume)
	}
	if bool(status.Mute) {
		t.Fatalf("Mute = %v; want false", status.Mute)
	}
	if status.DB != -58.5 {
		t.Fatalf("DB = %v; want -58.5", status.DB)
	}
	if status.ETag != "123" {
		t.Fatalf("ETag = %q; want 123", status.ETag)
	}
}

func TestVolumeSetRequest(t *testing.T) {
	t.Parallel()

	gotPath := make(chan string, 1)
	gotQuery := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath <- r.URL.Path
		gotQuery <- r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<volume/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.VolumeSet(context.Background(), VolumeSetOptions{Level: 10, TellSlaves: true}); err != nil {
		t.Fatalf("VolumeSet() err = %v", err)
	}

	if p := <-gotPath; p != "/Volume" {
		t.Fatalf("path = %q; want /Volume", p)
	}
	if q := <-gotQuery; q != "level=10&tell_slaves=1" {
		t.Fatalf("query = %q; want level=10&tell_slaves=1", q)
	}
}

func TestAddSlaveRequest(t *testing.T) {
	t.Parallel()

	got := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<addSlave/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.AddSlave(context.Background(), AddSlaveOptions{SlaveHost: "192.168.1.50", SlavePort: 11000, GroupName: "Kitchen"}); err != nil {
		t.Fatalf("AddSlave() err = %v", err)
	}

	if u := <-got; u != "/AddSlave?group=Kitchen&port=11000&slave=192.168.1.50" {
		t.Fatalf("url = %q; want /AddSlave?group=Kitchen&port=11000&slave=192.168.1.50", u)
	}
}

func TestRepeatRequest(t *testing.T) {
	t.Parallel()

	got := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<playlist/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.Repeat(context.Background(), 2); err != nil {
		t.Fatalf("Repeat() err = %v", err)
	}

	if u := <-got; u != "/Repeat?state=2" {
		t.Fatalf("url = %q; want /Repeat?state=2", u)
	}
}

func TestDryRunAllowsReadsBlocksWrites(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.Path
		switch r.URL.Path {
		case "/Status":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<status state="play" volume="15" mute="0"/>`))
		default:
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ok/>`))
		}
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{Timeout: 2 * time.Second, DryRun: true})

	if _, err := client.Status(context.Background(), StatusOptions{}); err != nil {
		t.Fatalf("Status() err = %v", err)
	}
	if p := <-seen; p != "/Status" {
		t.Fatalf("first request path = %q; want /Status", p)
	}

	err := client.Play(context.Background(), PlayOptions{})
	if !errors.Is(err, ErrDryRun) {
		t.Fatalf("Play() err = %v; want ErrDryRun", err)
	}
	select {
	case p := <-seen:
		t.Fatalf("unexpected request after dry-run write: %q", p)
	default:
	}
}
