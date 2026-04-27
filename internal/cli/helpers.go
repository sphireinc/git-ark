package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func writeFileExclusive(path string, content []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		}
	}
	return os.WriteFile(path, content, 0o644)
}

func absPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func printJSON(w interface{}, v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func cleanString(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// cmdOutLine writes a line to stdout and returns any write failure.
func cmdOutLine(cmd *cobra.Command, args ...any) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), args...)
	return err
}

// cmdOutf is the stdout version of fmt.Fprintf with error handling.
func cmdOutf(cmd *cobra.Command, format string, args ...any) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	return err
}

// cmdErrLine writes a line to stderr and returns any write failure.
func cmdErrLine(cmd *cobra.Command, args ...any) error {
	_, err := fmt.Fprintln(cmd.ErrOrStderr(), args...)
	return err
}

// cmdErrf is the stderr version of fmt.Fprintf with error handling.
func cmdErrf(cmd *cobra.Command, format string, args ...any) error {
	_, err := fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
	return err
}
