package git

import (
	"context"
	"fmt"
	"sort"
)

type RemoteStatus struct {
	Name   string
	URL    string
	Exists bool
}

func (c *Client) ListRemoteStatuses(ctx context.Context, repo string, expected map[string]string) ([]RemoteStatus, error) {
	remoteMap, err := c.RemoteMap(ctx, repo)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	statuses := make([]RemoteStatus, 0, len(names))
	for _, name := range names {
		url, ok := remoteMap[name]
		statuses = append(statuses, RemoteStatus{Name: name, URL: expected[name], Exists: ok})
		if ok {
			statuses[len(statuses)-1].URL = url
		}
	}
	return statuses, nil
}

func (c *Client) EnsureRemote(ctx context.Context, repo, name, url string, manage bool) (string, error) {
	if !manage {
		return url, nil
	}
	current, err := c.RemoteURL(ctx, repo, name)
	if err != nil {
		if err := c.AddRemote(ctx, repo, name, url); err != nil {
			return "", err
		}
		return url, nil
	}
	if current != url {
		if err := c.SetRemoteURL(ctx, repo, name, url); err != nil {
			return "", err
		}
	}
	return url, nil
}

func (c *Client) RemoteExists(ctx context.Context, repo, name string) (bool, error) {
	_, err := c.RemoteURL(ctx, repo, name)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *Client) EnsureRemotes(ctx context.Context, repo string, desired map[string]string, manage bool) ([]string, error) {
	addedOrUpdated := make([]string, 0, len(desired))
	for name, url := range desired {
		if _, err := c.EnsureRemote(ctx, repo, name, url, manage); err != nil {
			return addedOrUpdated, fmt.Errorf("sync remote %q: %w", name, err)
		}
		addedOrUpdated = append(addedOrUpdated, name)
	}
	sort.Strings(addedOrUpdated)
	return addedOrUpdated, nil
}
