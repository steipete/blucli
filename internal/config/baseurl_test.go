package config

import "testing"

func TestDeviceBaseURLDefaults(t *testing.T) {
	t.Parallel()

	u := (Device{}).BaseURL()
	if u.String() != "http://127.0.0.1:11000/" {
		t.Fatalf("url=%q", u.String())
	}
	u = (Device{Host: "192.0.2.1"}).BaseURL()
	if u.String() != "http://192.0.2.1:11000/" {
		t.Fatalf("url=%q", u.String())
	}
	u = (Device{Host: "192.0.2.1", Port: 12000}).BaseURL()
	if u.String() != "http://192.0.2.1:12000/" {
		t.Fatalf("url=%q", u.String())
	}
}
