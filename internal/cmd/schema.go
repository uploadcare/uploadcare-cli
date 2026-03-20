package cmd

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type apiSchema struct {
	Version     string       `json:"version"`
	GlobalFlags []flagSchema `json:"global_flags"`
	Commands    []cmdSchema  `json:"commands"`
	ExitCodes   map[string]string `json:"exit_codes"`
}

type flagSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type cmdSchema struct {
	Path     string       `json:"path"`
	Short    string       `json:"short"`
	Long     string       `json:"long,omitempty"`
	Args     argsSchema   `json:"args"`
	Flags    []flagSchema `json:"flags,omitempty"`
	Examples []string     `json:"examples,omitempty"`
}

type argsSchema struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func newAPISchemaCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "api-schema",
		Short: "Print machine-readable CLI command schema as JSON",
		Long: `Print the complete CLI command tree as a JSON schema.

This lets AI agents and scripts discover all available commands, flags,
arguments, and examples in a single call — no need to recursively
invoke --help on each subcommand.

Output includes: version, global flags, all commands with their flags
and examples, and exit code definitions.

No authentication required.`,
		Example: `  # Get the full schema
  uploadcare api-schema

  # Count available commands
  uploadcare api-schema | jq '.commands | length'

  # List all command paths
  uploadcare api-schema | jq -r '.commands[].path'`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			root := cmd.Root()
			schema := apiSchema{
				Version:     version,
				GlobalFlags: collectFlags(root.PersistentFlags()),
				Commands:    collectCommands(root, ""),
				ExitCodes: map[string]string{
					"0": "Success",
					"1": "API/runtime error",
					"2": "Usage error",
					"3": "Auth/config error",
				},
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			enc.Encode(schema)
		},
	}
}

func collectFlags(fs *pflag.FlagSet) []flagSchema {
	var flags []flagSchema
	fs.VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagSchema{
			Name:        f.Name,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})
	return flags
}

func collectCommands(cmd *cobra.Command, prefix string) []cmdSchema {
	var result []cmdSchema
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" || sub.Name() == "api-schema" {
			continue
		}

		path := sub.Name()
		if prefix != "" {
			path = prefix + " " + sub.Name()
		}

		cs := cmdSchema{
			Path:  path,
			Short: sub.Short,
			Long:  sub.Long,
			Args:  extractArgs(sub),
			Flags: collectFlags(sub.LocalFlags()),
		}

		if sub.Example != "" {
			cs.Examples = parseExamples(sub.Example)
		}

		// Only add leaf commands (those with RunE/Run) or parent commands
		if sub.HasSubCommands() {
			// Add the parent as a grouping entry
			result = append(result, cs)
			// Recurse into subcommands
			result = append(result, collectCommands(sub, path)...)
		} else {
			result = append(result, cs)
		}
	}
	return result
}

func extractArgs(cmd *cobra.Command) argsSchema {
	// Parse from Use string: "command <arg1> <arg2>..." or "command <arg>..."
	use := cmd.Use
	parts := strings.Fields(use)
	if len(parts) <= 1 {
		return argsSchema{Min: 0, Max: 0}
	}

	argParts := parts[1:]
	min := 0
	max := 0
	hasVariadic := false

	for _, p := range argParts {
		if strings.HasSuffix(p, "...") {
			hasVariadic = true
		}
		if strings.HasPrefix(p, "<") {
			min++
			max++
		} else if strings.HasPrefix(p, "[") {
			max++
		}
	}

	if hasVariadic {
		// ArbitraryArgs — set min to 0 for optional variadic
		if min > 0 {
			min-- // The variadic arg itself is optional beyond 0
		}
		max = -1 // unlimited
	}

	return argsSchema{Min: min, Max: max}
}

func parseExamples(example string) []string {
	var examples []string
	var current strings.Builder

	for _, line := range strings.Split(example, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Len() > 0 {
				examples = append(examples, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			if current.Len() > 0 {
				examples = append(examples, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}

		if current.Len() > 0 {
			current.WriteString(" ")
		}
		// Remove trailing backslash (line continuation)
		trimmed = strings.TrimSuffix(trimmed, "\\")
		trimmed = strings.TrimSpace(trimmed)
		current.WriteString(trimmed)
	}

	if current.Len() > 0 {
		examples = append(examples, strings.TrimSpace(current.String()))
	}

	return examples
}
