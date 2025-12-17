package config

import "testing"

func TestParseDevice_Errors(t *testing.T) {
	t.Parallel()

	if _, err := ParseDevice(""); err == nil {
		t.Fatalf("want error")
	}
	if _, err := ParseDevice("[]"); err == nil {
		t.Fatalf("want error")
	}
	if _, err := ParseDevice("127.0.0.1:nope"); err == nil {
		t.Fatalf("want error")
	}
	if _, err := ParseDevice("http://127.0.0.1:nope"); err == nil {
		t.Fatalf("want error")
	}
}
