package bluos

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/url"
	"strconv"
)

type BrowseOptions struct {
	Key                  string
	Q                    string
	WithContextMenuItems bool
}

type Browse struct {
	XMLName xml.Name `xml:"browse" json:"-"`
	SID     string   `xml:"sid,attr" json:"sid,omitempty"`
	Type    string   `xml:"type,attr" json:"type,omitempty"`

	Items []BrowseItem `xml:"item" json:"items,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

type BrowseItem struct {
	Text      string `xml:"text,attr" json:"text,omitempty"`
	Type      string `xml:"type,attr" json:"type,omitempty"`
	Image     string `xml:"image,attr" json:"image,omitempty"`
	BrowseKey string `xml:"browseKey,attr" json:"browseKey,omitempty"`
	PlayURL   string `xml:"playURL,attr" json:"playURL,omitempty"`
	AddURL    string `xml:"addURL,attr" json:"addURL,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) Browse(ctx context.Context, opts BrowseOptions) (Browse, error) {
	q := url.Values{}
	q.Set("key", opts.Key)
	if opts.Q != "" {
		q.Set("q", opts.Q)
	}
	if opts.WithContextMenuItems {
		q.Set("withContextMenuItems", "1")
	}

	data, err := c.getRead(ctx, "/Browse", q)
	if err != nil {
		return Browse{}, err
	}

	var browse Browse
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&browse); err != nil {
		return Browse{}, err
	}
	return browse, nil
}

type PlaylistsOptions struct {
	Service  string
	Category string
	Expr     string
}

type Playlists struct {
	XMLName xml.Name `xml:"playlists" json:"-"`
	Service string   `xml:"service,attr" json:"service,omitempty"`

	Names []PlaylistName `xml:"name" json:"names,omitempty"`
}

type PlaylistName struct {
	Text string `xml:",chardata" json:"name"`

	ID             string `xml:"id,attr" json:"id,omitempty"`
	Image          string `xml:"image,attr" json:"image,omitempty"`
	Description    string `xml:"description,attr" json:"description,omitempty"`
	Tracks         int    `xml:"tracks,attr" json:"tracks,omitempty"`
	TimeSeconds    int    `xml:"time,attr" json:"time,omitempty"`
	DeletePlaylist int    `xml:"deletePlaylist,attr" json:"deletePlaylist,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) Playlists(ctx context.Context, opts PlaylistsOptions) (Playlists, error) {
	q := url.Values{}
	if opts.Service != "" {
		q.Set("service", opts.Service)
	}
	if opts.Category != "" {
		q.Set("category", opts.Category)
	}
	if opts.Expr != "" {
		q.Set("expr", opts.Expr)
	}

	data, err := c.getRead(ctx, "/Playlists", q)
	if err != nil {
		return Playlists{}, err
	}

	var playlists Playlists
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&playlists); err != nil {
		return Playlists{}, err
	}
	return playlists, nil
}

type RadioBrowseOptions struct {
	Service string
	Expr    string
}

type RadioBrowse struct {
	XMLName xml.Name `xml:"radiotime" json:"-"`
	Service string   `xml:"service,attr" json:"service,omitempty"`

	Categories []RadioCategory `xml:"category" json:"categories,omitempty"`
	Items      []RadioItem     `xml:"item" json:"items,omitempty"`
}

type RadioCategory struct {
	Text string `xml:"text,attr" json:"text,omitempty"`

	Items []RadioItem `xml:"item" json:"items,omitempty"`
}

type RadioItem struct {
	ID        string `xml:"id,attr" json:"id,omitempty"`
	Text      string `xml:"text,attr" json:"text,omitempty"`
	Type      string `xml:"type,attr" json:"type,omitempty"`
	URL       string `xml:"URL,attr" json:"url,omitempty"`
	Image     string `xml:"image,attr" json:"image,omitempty"`
	InputType string `xml:"inputType,attr" json:"inputType,omitempty"`

	PlayerName string `xml:"playerName,attr" json:"playerName,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) RadioBrowse(ctx context.Context, opts RadioBrowseOptions) (RadioBrowse, error) {
	q := url.Values{}
	q.Set("service", opts.Service)
	if opts.Expr != "" {
		q.Set("expr", opts.Expr)
	}

	data, err := c.getRead(ctx, "/RadioBrowse", q)
	if err != nil {
		return RadioBrowse{}, err
	}

	var rb RadioBrowse
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&rb); err != nil {
		return RadioBrowse{}, err
	}
	return rb, nil
}

func (c *Client) Sleep(ctx context.Context) (int, error) {
	data, err := c.getWrite(ctx, "/Sleep", nil)
	if err != nil {
		return 0, err
	}
	s := string(bytes.TrimSpace(data))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}
