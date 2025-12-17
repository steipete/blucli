package app

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
	"github.com/steipete/blucli/internal/spotify"
)

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

	api := spotify.NewAPI(spotify.APIOptions{
		APIBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_API_BASE_URL")),
		HTTP:       http.DefaultClient,
	})

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

	oauth, err := spotify.NewOAuth(spotify.OAuthOptions{
		AccountsBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_ACCOUNTS_BASE_URL")),
		ClientID:        clientID,
		HTTP:            http.DefaultClient,
	})
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
