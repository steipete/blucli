package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
	"github.com/steipete/blucli/internal/spotify"
)

func cmdSpotify(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("spotify: missing subcommand (login|logout|open|devices|search|play)")
		return 2
	}

	switch args[0] {
	case "login":
		return cmdSpotifyLogin(ctx, out, paths, cfg, args[1:])
	case "logout":
		cfg.Spotify.Token = config.SpotifyToken{}
		if err := config.SaveConfig(paths.ConfigPath, cfg); err != nil {
			out.Errorf("spotify logout: %v", err)
			return 1
		}
		return 0
	case "open":
		device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
		if resolveErr != nil {
			out.Errorf("device: %v", resolveErr)
			return 1
		}
		client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
		if err := client.Play(ctx, bluos.PlayOptions{URL: "Spotify:play"}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("spotify open: %v", err)
			return 1
		}
		return 0
	case "devices":
		return cmdSpotifyDevices(ctx, out, paths, cfg, args[1:])
	case "search":
		return cmdSpotifySearch(ctx, out, paths, cfg, args[1:])
	case "play":
		return cmdSpotifyPlay(ctx, out, paths, cfg, cache, deviceArg, allowDiscover, discoverTimeout, httpTimeout, dryRun, trace, args[1:])
	default:
		out.Errorf("spotify: unknown subcommand %q (expected login|logout|open|devices|search|play)", args[0])
		return 2
	}
}

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

	oauth, err := spotify.NewOAuth(spotify.OAuthOptions{ClientID: clientID})
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

func cmdSpotifyDevices(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, args []string) int {
	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{})
	devs, err := api.Devices(ctx, accessToken)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}
	out.Print(devs)
	return 0
}

func cmdSpotifySearch(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, args []string) int {
	if len(args) == 0 {
		out.Errorf("spotify search: missing query")
		return 2
	}
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		out.Errorf("spotify search: missing query")
		return 2
	}

	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{})
	res, err := api.Search(ctx, accessToken, query, []string{"track", "artist"}, 5)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}
	out.Print(res)
	return 0
}

func cmdSpotifyPlay(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	flags := flag.NewFlagSet("spotify play", flag.ContinueOnError)
	flags.SetOutput(out.Stderr())

	var playType string
	var pick int
	var market string
	var wait time.Duration
	var spotifyDeviceID string
	var noActivate bool

	flags.StringVar(&playType, "type", "auto", "pick type: auto|artist|track")
	flags.IntVar(&pick, "pick", 0, "pick nth result (0-based within chosen type)")
	flags.StringVar(&market, "market", "US", "market for artist top tracks (e.g. US)")
	flags.DurationVar(&wait, "wait", 12*time.Second, "wait for Spotify Connect device to appear")
	flags.StringVar(&spotifyDeviceID, "spotify-device", "", "Spotify Connect device id (optional override)")
	flags.BoolVar(&noActivate, "no-activate", false, "don't call Spotify:play on BluOS before controlling Spotify")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) == 0 {
		out.Errorf("spotify play: missing query")
		return 2
	}
	query := strings.TrimSpace(strings.Join(rest, " "))
	if query == "" {
		out.Errorf("spotify play: missing query")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}

	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg)
	if err != nil {
		out.Errorf("spotify play: %v", err)
		return 1
	}

	playerName := ""
	if spotifyDeviceID == "" {
		st, err := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: true, Trace: nil}).Status(ctx, bluos.StatusOptions{})
		if err == nil {
			playerName = strings.TrimSpace(st.Name)
		}
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
	if !noActivate {
		if err := client.Play(ctx, bluos.PlayOptions{URL: "Spotify:play"}); err != nil && !errors.Is(err, bluos.ErrDryRun) {
			out.Errorf("spotify play: activate Spotify: %v", err)
			return 1
		}
	}

	api := spotify.NewAPI(spotify.APIOptions{})

	if spotifyDeviceID == "" {
		deadline := time.Now().Add(wait)
		for {
			devs, err := api.Devices(ctx, accessToken)
			if err == nil {
				if d, ok := matchSpotifyDevice(devs.Devices, playerName); ok {
					spotifyDeviceID = d.ID
					break
				}
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(1200 * time.Millisecond)
		}
	}

	if strings.TrimSpace(spotifyDeviceID) == "" {
		devs, err := api.Devices(ctx, accessToken)
		if err != nil {
			out.Errorf("spotify play: list devices: %v", err)
			return 1
		}
		out.Errorf("spotify play: unable to pick Spotify Connect device (use --spotify-device). candidates=%d", len(devs.Devices))
		out.Print(devs)
		return 1
	}

	res, err := api.Search(ctx, accessToken, query, []string{"track", "artist"}, 10)
	if err != nil {
		out.Errorf("spotify play: search: %v", err)
		return 1
	}

	chosenType := strings.TrimSpace(strings.ToLower(playType))
	if chosenType == "" {
		chosenType = "auto"
	}

	if chosenType == "auto" {
		chosenType = autoPickType(query, res)
	}

	switch chosenType {
	case "track":
		if pick < 0 || pick >= len(res.Tracks.Items) {
			out.Errorf("spotify play: pick out of range for tracks (0..%d): %d", max0(len(res.Tracks.Items)-1), pick)
			return 2
		}
		if len(res.Tracks.Items) == 0 {
			out.Errorf("spotify play: no track results")
			return 1
		}
		uri := strings.TrimSpace(res.Tracks.Items[pick].URI)
		if uri == "" {
			out.Errorf("spotify play: missing track uri")
			return 1
		}
		_ = api.Transfer(ctx, accessToken, spotifyDeviceID, false) // best-effort; some accounts/devices behave differently.
		if err := api.Play(ctx, accessToken, spotifyDeviceID, spotify.PlayRequest{URIs: []string{uri}}); err != nil {
			out.Errorf("spotify play: %v", err)
			return 1
		}
		out.Print(map[string]any{"type": "track", "uri": uri})
		return 0
	case "artist":
		if pick < 0 || pick >= len(res.Artists.Items) {
			out.Errorf("spotify play: pick out of range for artists (0..%d): %d", max0(len(res.Artists.Items)-1), pick)
			return 2
		}
		if len(res.Artists.Items) == 0 {
			out.Errorf("spotify play: no artist results")
			return 1
		}
		artist := res.Artists.Items[pick]
		if strings.TrimSpace(artist.ID) == "" {
			out.Errorf("spotify play: missing artist id")
			return 1
		}
		top, err := api.ArtistTopTracks(ctx, accessToken, artist.ID, market)
		if err != nil {
			out.Errorf("spotify play: artist top tracks: %v", err)
			return 1
		}
		if len(top.Tracks) == 0 {
			out.Errorf("spotify play: artist has no top tracks")
			return 1
		}
		var uris []string
		for _, t := range top.Tracks {
			if u := strings.TrimSpace(t.URI); u != "" {
				uris = append(uris, u)
			}
		}
		if len(uris) == 0 {
			out.Errorf("spotify play: artist top tracks missing uris")
			return 1
		}
		_ = api.Transfer(ctx, accessToken, spotifyDeviceID, false)
		if err := api.Play(ctx, accessToken, spotifyDeviceID, spotify.PlayRequest{URIs: uris}); err != nil {
			out.Errorf("spotify play: %v", err)
			return 1
		}
		out.Print(map[string]any{"type": "artist", "artist": artist.Name, "count": len(uris)})
		return 0
	default:
		out.Errorf("spotify play: unknown --type %q (expected auto|artist|track)", playType)
		return 2
	}
}

func matchSpotifyDevice(devices []spotify.Device, playerName string) (spotify.Device, bool) {
	if len(devices) == 0 {
		return spotify.Device{}, false
	}
	name := strings.ToLower(strings.TrimSpace(playerName))
	if name != "" {
		for _, d := range devices {
			if strings.ToLower(strings.TrimSpace(d.Name)) == name {
				return d, true
			}
		}
		for _, d := range devices {
			if strings.Contains(strings.ToLower(strings.TrimSpace(d.Name)), name) {
				return d, true
			}
		}
	}
	for _, d := range devices {
		if d.IsActive {
			return d, true
		}
	}
	if len(devices) == 1 {
		return devices[0], true
	}
	return spotify.Device{}, false
}

func autoPickType(query string, res spotify.SearchResponse) string {
	q := normalizeName(query)
	for _, a := range res.Artists.Items {
		if normalizeName(a.Name) == q && q != "" {
			return "artist"
		}
	}
	if len(res.Tracks.Items) > 0 {
		return "track"
	}
	if len(res.Artists.Items) > 0 {
		return "artist"
	}
	return "track"
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "  ", " ")
	return s
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func spotifyAccessToken(ctx context.Context, paths config.PathSet, cfg config.Config) (string, config.Config, error) {
	clientID := strings.TrimSpace(cfg.Spotify.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	}
	if clientID == "" {
		return "", cfg, errors.New("missing spotify client id (run `blu spotify login` or set SPOTIFY_CLIENT_ID)")
	}

	tok := cfg.Spotify.Token
	stored := spotify.Token{
		AccessToken:  strings.TrimSpace(tok.AccessToken),
		RefreshToken: strings.TrimSpace(tok.RefreshToken),
		ExpiresAt:    tok.ExpiresAt,
		TokenType:    strings.TrimSpace(tok.TokenType),
		Scope:        strings.TrimSpace(tok.Scope),
	}
	if stored.AccessToken == "" {
		return "", cfg, errors.New("missing spotify token (run `blu spotify login`)")
	}

	if !spotify.TokenExpired(stored, 45*time.Second) {
		return stored.AccessToken, cfg, nil
	}
	if strings.TrimSpace(stored.RefreshToken) == "" {
		return "", cfg, errors.New("spotify token expired and missing refresh token (run `blu spotify login`)")
	}

	oauth, err := spotify.NewOAuth(spotify.OAuthOptions{ClientID: clientID})
	if err != nil {
		return "", cfg, err
	}
	refreshed, err := oauth.Refresh(ctx, stored.RefreshToken)
	if err != nil {
		return "", cfg, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = stored.RefreshToken
	}
	cfg.Spotify.ClientID = clientID
	cfg.Spotify.Token = config.SpotifyToken{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		ExpiresAt:    refreshed.ExpiresAt,
		TokenType:    refreshed.TokenType,
		Scope:        refreshed.Scope,
	}
	if err := config.SaveConfig(paths.ConfigPath, cfg); err != nil {
		return "", cfg, err
	}
	return refreshed.AccessToken, cfg, nil
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
