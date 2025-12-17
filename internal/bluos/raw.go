package bluos

import (
	"context"
	"net/url"
)

func (c *Client) RawGet(ctx context.Context, path string, params map[string]string, mutating bool) ([]byte, error) {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	if mutating {
		return c.getWrite(ctx, path, q)
	}
	return c.getRead(ctx, path, q)
}
