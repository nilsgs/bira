package cmd

import (
	"encoding/json"
	"fmt"
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

// printJSON writes v as JSON to stdout. Used by all commands when --json is set.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding json:", err)
		os.Exit(1)
	}
}

// printTable writes rows in aligned columns to stdout using tabwriter.
func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Repeat("─\t", len(headers)))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// exitNotFound prints a not-found error and exits with code 2.
func exitNotFound(entity, id string) {
	fmt.Fprintf(os.Stderr, "%s not found: %s\n", entity, id)
	os.Exit(2)
}

// output dispatches to JSON or human-readable depending on --json flag.
func output(v any, humanFn func()) {
	if jsonOutput {
		printJSON(v)
	} else {
		humanFn()
	}
}
