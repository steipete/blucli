package bluos

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
)

type Presets struct {
	XMLName xml.Name `xml:"presets" json:"-"`
	PrID    string   `xml:"prid,attr" json:"prid,omitempty"`

	Presets []Preset `xml:"preset" json:"presets"`
}

type Preset struct {
	ID    int    `xml:"id,attr" json:"id"`
	Name  string `xml:"name,attr" json:"name,omitempty"`
	Image string `xml:"image,attr" json:"image,omitempty"`
	URL   string `xml:"url,attr" json:"url,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) Presets(ctx context.Context) (Presets, error) {
	data, err := c.getRead(ctx, "/Presets", nil)
	if err != nil {
		return Presets{}, err
	}

	var presets Presets
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&presets); err != nil {
		return Presets{}, err
	}
	return presets, nil
}

type LoadedResponse struct {
	XMLName xml.Name `xml:"loaded" json:"-"`

	Service string `xml:"service,attr" json:"service,omitempty"`
	Entries int    `xml:"entries" json:"entries,omitempty"`
}

func (c *Client) LoadPreset(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("missing id")
	}
	q := url.Values{}
	q.Set("id", id)

	data, err := c.getWrite(ctx, "/Preset", q)
	if err != nil {
		return nil, err
	}

	var loaded LoadedResponse
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&loaded); err == nil && loaded.XMLName.Local != "" {
		return loaded, nil
	}

	var presets Presets
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&presets); err == nil && presets.XMLName.Local != "" {
		return presets, nil
	}

	return map[string]string{"xml": string(data)}, nil
}
