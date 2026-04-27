package git

import (
	"context"
	"strings"
)

type RemoteHealth struct {
	URL       string
	Reachable bool
	RefCount  int
	Error     string
	Hint      string
	Empty     bool
}

func (c *Client) ProbeRemote(ctx context.Context, url string) RemoteHealth {
	health := RemoteHealth{URL: url}
	res, err := c.run(ctx, "", "ls-remote", "--heads", "--tags", url)
	if err != nil {
		health.Reachable = false
		health.Error = strings.TrimSpace(err.Error())
		health.Hint = GitErrorHint(err, res.Stderr)
		return health
	}
	health.Reachable = true
	health.RefCount = len(splitNonEmptyLines(res.Stdout))
	health.Empty = health.RefCount == 0
	return health
}
