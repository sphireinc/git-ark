package git

import (
	"context"
	"strings"
)

func (c *Client) StatusSummary(ctx context.Context, repo string) (string, error) {
	res, err := c.run(ctx, repo, "status", "--short")
	if err != nil {
		return "", err
	}
	return res.Stdout, nil
}

func (c *Client) HasNotes(ctx context.Context, repo string) (bool, error) {
	res, err := c.run(ctx, repo, "show-ref", "--heads", "--tags")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) != "", nil
}
