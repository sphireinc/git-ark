package backup

import (
	"time"

	"git-ark/internal/git"
)

type Service struct {
	Git *git.Client
	Now func() time.Time
}

func New(g *git.Client) *Service {
	return &Service{
		Git: g,
		Now: time.Now,
	}
}
