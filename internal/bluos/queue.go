package bluos

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
)

type PlaylistOptions struct {
	Start *int
	End   *int
}

type Playlist struct {
	XMLName  xml.Name `xml:"playlist" json:"-"`
	Name     string   `xml:"name,attr" json:"name,omitempty"`
	Modified int      `xml:"modified,attr" json:"modified,omitempty"`
	Length   int      `xml:"length,attr" json:"length,omitempty"`
	ID       int      `xml:"id,attr" json:"id,omitempty"`
	Shuffle  int      `xml:"shuffle,attr" json:"shuffle,omitempty"`
	Repeat   int      `xml:"repeat,attr" json:"repeat,omitempty"`

	Songs []PlaylistSong `xml:"song" json:"songs,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

type PlaylistSong struct {
	ID      int    `xml:"id,attr" json:"id"`
	Service string `xml:"service,attr" json:"service,omitempty"`
	SongID  string `xml:"songid,attr" json:"songid,omitempty"`

	Title   string `xml:"title" json:"title,omitempty"`
	Artist  string `xml:"art" json:"artist,omitempty"`
	Album   string `xml:"alb" json:"album,omitempty"`
	Fn      string `xml:"fn" json:"fn,omitempty"`
	Quality string `xml:"quality" json:"quality,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) Playlist(ctx context.Context, opts PlaylistOptions) (Playlist, error) {
	q := url.Values{}
	if opts.Start != nil {
		q.Set("start", strconv.Itoa(*opts.Start))
	}
	if opts.End != nil {
		q.Set("end", strconv.Itoa(*opts.End))
	}

	data, err := c.getRead(ctx, "/Playlist", q)
	if err != nil {
		return Playlist{}, err
	}

	var pl Playlist
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (c *Client) Clear(ctx context.Context) (Playlist, error) {
	data, err := c.getWrite(ctx, "/Clear", nil)
	if err != nil {
		return Playlist{}, err
	}

	var pl Playlist
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (c *Client) Delete(ctx context.Context, id int) (Playlist, error) {
	if id < 0 {
		return Playlist{}, fmt.Errorf("id out of range: %d", id)
	}
	q := url.Values{}
	q.Set("id", strconv.Itoa(id))

	data, err := c.getWrite(ctx, "/Delete", q)
	if err != nil {
		return Playlist{}, err
	}

	var pl Playlist
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (c *Client) Move(ctx context.Context, oldID, newID int) (Playlist, error) {
	if oldID < 0 || newID < 0 {
		return Playlist{}, fmt.Errorf("id out of range: old=%d new=%d", oldID, newID)
	}
	q := url.Values{}
	q.Set("old", strconv.Itoa(oldID))
	q.Set("new", strconv.Itoa(newID))

	data, err := c.getWrite(ctx, "/Move", q)
	if err != nil {
		return Playlist{}, err
	}

	var pl Playlist
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

type SaveResponse struct {
	XMLName xml.Name `xml:"saved" json:"-"`
	Entries int      `xml:"entries" json:"entries"`
}

func (c *Client) Save(ctx context.Context, name string) (SaveResponse, error) {
	if name == "" {
		return SaveResponse{}, fmt.Errorf("missing name")
	}
	q := url.Values{}
	q.Set("name", name)

	data, err := c.getWrite(ctx, "/Save", q)
	if err != nil {
		return SaveResponse{}, err
	}

	var resp SaveResponse
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&resp); err != nil {
		return SaveResponse{}, err
	}
	return resp, nil
}
