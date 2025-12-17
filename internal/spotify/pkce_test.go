package spotify

import "testing"

func TestCodeChallengeS256(t *testing.T) {
	t.Parallel()

	// RFC 7636 example input/output
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if got := CodeChallengeS256(verifier); got != want {
		t.Fatalf("challenge = %q; want %q", got, want)
	}
}
