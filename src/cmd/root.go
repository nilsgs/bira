package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	// Set via -ldflags at build time.
	version = "dev"
	commit  = "none"

	jsonOutput  bool
	projectFlag string
)

var rootCmd = &cobra.Command{
	Use:   "bira",
	Short: "AI-agent optimised project management CLI",
	Long:  "bira tracks projects, features, and tasks. Designed for AI agents with structured JSON output.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		var nfe *notFoundError
		if errors.As(err, &nfe) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version + "+" + commit
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output JSON instead of table format")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "override project context (default: read from .bira)")
}

// --- output helpers ---

// printJSON writes v as JSON to w. Used by all commands when --json is set.
func printJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding json:", err)
		os.Exit(1)
	}
}

// printTable writes rows in aligned columns to w using tabwriter.
func printTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	fmt.Fprintln(tw, strings.Repeat("─\t", len(headers)))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// notFoundError is returned when a requested entity does not exist.
// Execute() uses this type to emit exit code 2 instead of the default 1.
type notFoundError struct {
	entity string
	id     string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.entity, e.id)
}

// notFoundErr returns a *notFoundError for the given entity and id.
func notFoundErr(entity, id string) error {
	return &notFoundError{entity: entity, id: id}
}



// output dispatches to JSON or human-readable depending on --json flag.
// The writer is obtained from cmd.OutOrStdout().
func output(cmd *cobra.Command, v any, humanFn func(io.Writer)) {
	w := cmd.OutOrStdout()
	if jsonOutput {
		printJSON(w, v)
	} else {
		humanFn(w)
	}
}
