package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
	"github.com/steipete/blucli/internal/spotify"
)

func cmdSpotifyLogin(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, args []string) int {
	flags := flag.NewFlagSet("spotify login", flag.ContinueOnError)
	flags.SetOutput(out.Stderr())

	var clientID string
	var redirect string
	var noOpen bool
	flags.StringVar(&clientID, "client-id", "", "Spotify app client id (or SPOTIFY_CLIENT_ID)")
	flags.StringVar(&redirect, "redirect", "http://127.0.0.1:8974/callback", "redirect URL (must be allowed in Spotify app settings)")
	flags.BoolVar(&noOpen, "no-open", false, "don't open browser, just print URL")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	}
	if clientID == "" {
		clientID = strings.TrimSpace(cfg.Spotify.ClientID)
	}
	if clientID == "" {
		out.Errorf("spotify login: missing client id (set --client-id or SPOTIFY_CLIENT_ID)")
		return 2
	}

	redirectURL, err := url.Parse(strings.TrimSpace(redirect))
	if err != nil || redirectURL.Scheme != "http" || redirectURL.Host == "" {
		out.Errorf("spotify login: invalid redirect url: %q", redirect)
		return 2
	}

	codeVerifier, err := spotify.NewCodeVerifier()
	if err != nil {
		out.Errorf("spotify login: %v", err)
		return 1
	}
	codeChallenge := spotify.CodeChallengeS256(codeVerifier)
	state, err := spotify.NewCodeVerifier()
	if err != nil {
		out.Errorf("spotify login: %v", err)
		return 1
	}

	ln, err := net.Listen("tcp", redirectURL.Host)
	if err != nil {
		out.Errorf("spotify login: listen %s: %v", redirectURL.Host, err)
		return 1
	}
	defer ln.Close()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("error") != "" {
			errCh <- fmt.Errorf("authorize: %s", q.Get("error"))
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Authorization failed. You can close this tab.")
			return
		}
		if q.Get("state") != state {
			errCh <- errors.New("authorize: state mismatch")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Authorization failed (state mismatch). You can close this tab.")
			return
		}
		code := strings.TrimSpace(q.Get("code"))
		if code == "" {
			errCh <- errors.New("authorize: missing code")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Authorization failed (missing code). You can close this tab.")
			return
		}
		codeCh <- code
		fmt.Fprintln(w, "OK. You can close this tab and return to the terminal.")
	})

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()

	scopes := strings.Join([]string{
		"user-read-playback-state",
		"user-modify-playback-state",
		"user-read-currently-playing",
	}, " ")

	authQ := url.Values{}
	authQ.Set("client_id", clientID)
	authQ.Set("response_type", "code")
	authQ.Set("redirect_uri", redirectURL.String())
	authQ.Set("code_challenge_method", "S256")
	authQ.Set("code_challenge", codeChallenge)
	authQ.Set("scope", scopes)
	authQ.Set("state", state)

	authURL := "https://accounts.spotify.com/authorize?" + authQ.Encode()
	fmt.Fprintln(out.Stdout(), authURL)
	if !noOpen {
		_ = openBrowser(authURL)
	}

	var code string
	select {
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		out.Errorf("spotify login: cancelled")
		return 1
	case err := <-errCh:
		_ = srv.Shutdown(context.Background())
		out.Errorf("spotify login: %v", err)
		return 1
	case code = <-codeCh:
		_ = srv.Shutdown(context.Background())
	}

	oauth, err := spotify.NewOAuth(spotify.OAuthOptions{
		AccountsBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_ACCOUNTS_BASE_URL")),
		ClientID:        clientID,
		HTTP:            http.DefaultClient,
	})
	if err != nil {
		out.Errorf("spotify login: %v", err)
		return 1
	}

	tok, err := oauth.ExchangeAuthorizationCode(ctx, code, redirectURL.String(), codeVerifier)
	if err != nil {
		out.Errorf("spotify login: %v", err)
		return 1
	}

	cfg.Spotify.ClientID = clientID
	cfg.Spotify.Token = config.SpotifyToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    tok.ExpiresAt,
		TokenType:    tok.TokenType,
		Scope:        tok.Scope,
	}
	if err := config.SaveConfig(paths.ConfigPath, cfg); err != nil {
		out.Errorf("spotify login: %v", err)
		return 1
	}
	return 0
}

func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}
