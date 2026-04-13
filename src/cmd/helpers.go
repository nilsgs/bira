package cmd

import "strings"

// parseTags splits a comma-separated tag string into a slice.
// Empty string returns nil.
func parseTags(s string) []string {
if s == "" {
return nil
}
parts := strings.Split(s, ",")
tags := make([]string, 0, len(parts))
for _, p := range parts {
if t := strings.TrimSpace(p); t != "" {
tags = append(tags, t)
}
}
return tags
}
