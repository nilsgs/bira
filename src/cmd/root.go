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
version = "dev"
commit  = "none"
)

func NewRootCmd() *cobra.Command {
root := &cobra.Command{
Use:           "bira",
Short:         "AI-agent optimised project management CLI",
Long:          "bira tracks projects, features, and tasks. Designed for AI agents with structured JSON output.",
SilenceUsage:  true,
SilenceErrors: true,
}
root.Version = version + "+" + commit
root.SetVersionTemplate("{{.Version}}\n")
root.PersistentFlags().Bool("json", false, "output JSON instead of table format")
root.PersistentFlags().String("project", "", "override project context (default: read from .bira)")

root.AddCommand(
newInitCmd(),
newContextCmd(),
newProjectCmd(),
newIdeaCmd(),
newBugCmd(),
newFeatureCmd(),
newMCPCmd(),
)
return root
}

func Execute() {
if err := NewRootCmd().Execute(); err != nil {
fmt.Fprintln(os.Stderr, "error:", err)
var nfe *notFoundError
if errors.As(err, &nfe) {
os.Exit(2)
}
os.Exit(1)
}
}

func printJSON(w io.Writer, v any) {
enc := json.NewEncoder(w)
enc.SetIndent("", "  ")
if err := enc.Encode(v); err != nil {
fmt.Fprintln(os.Stderr, "error encoding json:", err)
os.Exit(1)
}
}

func printTable(w io.Writer, headers []string, rows [][]string) {
tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
fmt.Fprintln(tw, strings.Join(headers, "\t"))
fmt.Fprintln(tw, strings.Repeat("─\t", len(headers)))
for _, row := range rows {
fmt.Fprintln(tw, strings.Join(row, "\t"))
}
tw.Flush()
}

type notFoundError struct {
entity string
id     string
}

func (e *notFoundError) Error() string {
return fmt.Sprintf("%s not found: %s", e.entity, e.id)
}

func notFoundErr(entity, id string) error {
return &notFoundError{entity: entity, id: id}
}

func output(cmd *cobra.Command, v any, humanFn func(io.Writer)) {
w := cmd.OutOrStdout()
jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
if jsonOut {
printJSON(w, v)
} else {
humanFn(w)
}
}
