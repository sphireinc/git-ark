package app

import (
	"fmt"
	"os"

	"git-ark/internal/cli"
)

type App struct {
	Root *cli.Root
}

func MustNew() *App {
	return &App{Root: cli.NewRoot()}
}

func (a *App) Execute() int {
	if a == nil || a.Root == nil {
		fmt.Fprintln(os.Stderr, "git-ark: application not initialized")
		return 1
	}
	if err := a.Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
