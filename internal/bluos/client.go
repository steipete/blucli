package bluos

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrDryRun = errors.New("dry-run")

type Options struct {
	Timeout time.Duration
	DryRun  bool
	Trace   io.Writer
}

type Client struct {
	baseURL *url.URL
	client  *http.Client
	dryRun  bool
	trace   io.Writer
}

func NewClient(baseURL *url.URL, opts Options) *Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		dryRun: opts.DryRun,
		trace:  opts.Trace,
	}
}

type StatusOptions struct {
	TimeoutSeconds int
	ETag           string
}

type Status struct {
	XMLName xml.Name `xml:"status" json:"-"`

	State string `xml:"state,attr" json:"state,omitempty"`
	Name  string `xml:"name,attr" json:"name,omitempty"`
	Model string `xml:"model,attr" json:"model,omitempty"`

	Volume int     `xml:"volume,attr" json:"volume"`
	DB     float64 `xml:"db,attr" json:"db,omitempty"`
	Mute   BoolInt `xml:"mute,attr" json:"mute"`

	Secs int `xml:"secs,attr" json:"secs,omitempty"`

	Title  string `xml:"title1,attr" json:"title,omitempty"`
	Artist string `xml:"artist,attr" json:"artist,omitempty"`
	Album  string `xml:"album,attr" json:"album,omitempty"`

	ETag string `xml:"etag,attr" json:"etag,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

func (c *Client) Status(ctx context.Context, opts StatusOptions) (Status, error) {
	q := url.Values{}
	if opts.TimeoutSeconds > 0 {
		q.Set("timeout", strconv.Itoa(opts.TimeoutSeconds))
	}
	if opts.ETag != "" {
		q.Set("etag", opts.ETag)
	}

	data, err := c.getRead(ctx, "/Status", q)
	if err != nil {
		return Status{}, err
	}

	var status Status
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&status); err != nil {
		return Status{}, err
	}
	return status, nil
}

type SyncStatusOptions struct {
	TimeoutSeconds int
	ETag           string
}

type SyncStatus struct {
	XMLName xml.Name `xml:"SyncStatus" json:"-"`

	ID      string `xml:"id,attr" json:"id,omitempty"`
	Name    string `xml:"name,attr" json:"name,omitempty"`
	Model   string `xml:"model,attr" json:"model,omitempty"`
	Group   string `xml:"group,attr" json:"group,omitempty"`
	Version string `xml:"schemaVersion,attr" json:"schemaVersion,omitempty"`

	Volume int     `xml:"volume,attr" json:"volume"`
	DB     float64 `xml:"db,attr" json:"db,omitempty"`
	Mute   BoolInt `xml:"mute,attr" json:"mute"`

	ETag string `xml:"etag,attr" json:"etag,omitempty"`

	Master *SyncMaster `xml:"master" json:"master,omitempty"`
	Slaves []SyncSlave `xml:"slave" json:"slaves,omitempty"`

	AnyAttrs []xml.Attr `xml:",any,attr" json:"-"`
}

type SyncMaster struct {
	Host string `xml:",chardata" json:"host"`
	Port int    `xml:"port,attr" json:"port"`
}

type SyncSlave struct {
	ID   string `xml:"id,attr" json:"id"`
	Port int    `xml:"port,attr" json:"port"`
}

func (c *Client) SyncStatus(ctx context.Context, opts SyncStatusOptions) (SyncStatus, error) {
	q := url.Values{}
	if opts.TimeoutSeconds > 0 {
		q.Set("timeout", strconv.Itoa(opts.TimeoutSeconds))
	}
	if opts.ETag != "" {
		q.Set("etag", opts.ETag)
	}

	data, err := c.getRead(ctx, "/SyncStatus", q)
	if err != nil {
		return SyncStatus{}, err
	}

	var sync SyncStatus
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&sync); err != nil {
		return SyncStatus{}, err
	}
	if sync.Master != nil {
		sync.Master.Host = strings.TrimSpace(sync.Master.Host)
	}
	return sync, nil
}

type PlayOptions struct {
	SeekSeconds int
	ID          int
	URL         string
}

func (c *Client) Play(ctx context.Context, opts PlayOptions) error {
	q := url.Values{}
	if opts.SeekSeconds > 0 {
		q.Set("seek", strconv.Itoa(opts.SeekSeconds))
	}
	if opts.ID > 0 {
		q.Set("id", strconv.Itoa(opts.ID))
	}
	if opts.URL != "" {
		q.Set("url", opts.URL)
	}
	_, err := c.getWrite(ctx, "/Play", q)
	return err
}

type PauseOptions struct {
	Toggle bool
}

func (c *Client) Pause(ctx context.Context, opts PauseOptions) error {
	q := url.Values{}
	if opts.Toggle {
		q.Set("toggle", "1")
	}
	_, err := c.getWrite(ctx, "/Pause", q)
	return err
}

func (c *Client) Stop(ctx context.Context) error {
	_, err := c.getWrite(ctx, "/Stop", nil)
	return err
}

func (c *Client) Skip(ctx context.Context) error {
	_, err := c.getWrite(ctx, "/Skip", nil)
	return err
}

func (c *Client) Back(ctx context.Context) error {
	_, err := c.getWrite(ctx, "/Back", nil)
	return err
}

func (c *Client) Shuffle(ctx context.Context, enabled bool) error {
	q := url.Values{}
	if enabled {
		q.Set("state", "1")
	} else {
		q.Set("state", "0")
	}
	_, err := c.getWrite(ctx, "/Shuffle", q)
	return err
}

func (c *Client) Repeat(ctx context.Context, state int) error {
	if state < 0 || state > 2 {
		return fmt.Errorf("repeat state out of range: %d", state)
	}
	q := url.Values{}
	q.Set("state", strconv.Itoa(state))
	_, err := c.getWrite(ctx, "/Repeat", q)
	return err
}

type VolumeSetOptions struct {
	Level      int
	TellSlaves bool
}

func (c *Client) VolumeSet(ctx context.Context, opts VolumeSetOptions) error {
	if opts.Level < 0 || opts.Level > 100 {
		return fmt.Errorf("level out of range: %d", opts.Level)
	}
	q := url.Values{}
	q.Set("level", strconv.Itoa(opts.Level))
	if opts.TellSlaves {
		q.Set("tell_slaves", "1")
	}
	_, err := c.getWrite(ctx, "/Volume", q)
	return err
}

type VolumeDeltaDBOptions struct {
	DeltaDB    int
	TellSlaves bool
}

func (c *Client) VolumeDeltaDB(ctx context.Context, opts VolumeDeltaDBOptions) error {
	if opts.DeltaDB == 0 {
		return nil
	}
	q := url.Values{}
	q.Set("db", strconv.Itoa(opts.DeltaDB))
	if opts.TellSlaves {
		q.Set("tell_slaves", "1")
	}
	_, err := c.getWrite(ctx, "/Volume", q)
	return err
}

type VolumeMuteOptions struct {
	Mute       bool
	TellSlaves bool
}

func (c *Client) VolumeMute(ctx context.Context, opts VolumeMuteOptions) error {
	q := url.Values{}
	if opts.Mute {
		q.Set("mute", "1")
	} else {
		q.Set("mute", "0")
	}
	if opts.TellSlaves {
		q.Set("tell_slaves", "1")
	}
	_, err := c.getWrite(ctx, "/Volume", q)
	return err
}

type AddSlaveOptions struct {
	SlaveHost string
	SlavePort int
	GroupName string
}

func (c *Client) AddSlave(ctx context.Context, opts AddSlaveOptions) error {
	if opts.SlaveHost == "" {
		return errors.New("missing slave host")
	}
	if opts.SlavePort == 0 {
		opts.SlavePort = 11000
	}
	q := url.Values{}
	q.Set("slave", opts.SlaveHost)
	q.Set("port", strconv.Itoa(opts.SlavePort))
	if opts.GroupName != "" {
		q.Set("group", opts.GroupName)
	}
	_, err := c.getWrite(ctx, "/AddSlave", q)
	return err
}

type RemoveSlaveOptions struct {
	SlaveHost string
	SlavePort int
}

func (c *Client) RemoveSlave(ctx context.Context, opts RemoveSlaveOptions) error {
	if opts.SlaveHost == "" {
		return errors.New("missing slave host")
	}
	if opts.SlavePort == 0 {
		opts.SlavePort = 11000
	}
	q := url.Values{}
	q.Set("slave", opts.SlaveHost)
	q.Set("port", strconv.Itoa(opts.SlavePort))
	_, err := c.getWrite(ctx, "/RemoveSlave", q)
	return err
}

func (c *Client) getRead(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.get(ctx, path, query, false)
}

func (c *Client) getWrite(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.get(ctx, path, query, true)
}

func (c *Client) get(ctx context.Context, path string, query url.Values, mutating bool) ([]byte, error) {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.trace != nil {
		fmt.Fprintf(c.trace, "http: GET %s\n", u.String())
	}
	if mutating && c.dryRun {
		return nil, ErrDryRun
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(data))
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	return data, nil
}

type BoolInt bool

func (b *BoolInt) UnmarshalXMLAttr(attr xml.Attr) error {
	v := strings.TrimSpace(strings.ToLower(attr.Value))
	switch v {
	case "1", "true", "yes", "on":
		*b = true
	case "0", "false", "no", "off", "":
		*b = false
	default:
		return fmt.Errorf("invalid bool: %q", attr.Value)
	}
	return nil
}
