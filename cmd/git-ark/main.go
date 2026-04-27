package main

import (
	"os"

	"git-ark/internal/app"
)

func main() {
	os.Exit(app.MustNew().Execute())
}
