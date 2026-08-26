package cmd

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type apiSchema struct {
	Version     string            `json:"version"`
	GlobalFlags []flagSchema      `json:"global_flags"`
	RootFlags   []flagSchema      `json:"root_flags,omitempty"`
	Commands    []cmdSchema       `json:"commands"`
	ExitCodes   map[string]string `json:"exit_codes"`
	AgentNotes  []string          `json:"agent_notes"`
	URLAPI      *urlAPISchema     `json:"url_api"`
}

type flagSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type cmdSchema struct {
	Path       string       `json:"path"`
	Short      string       `json:"short"`
	Long       string       `json:"long,omitempty"`
	Args       argsSchema   `json:"args"`
	Flags      []flagSchema `json:"flags,omitempty"`
	Examples   []string     `json:"examples,omitempty"`
	JSONFields []string     `json:"json_fields,omitempty"`
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
				RootFlags:   rootOnlyFlags(root),
				Commands:    collectCommands(root, ""),
				ExitCodes: map[string]string{
					"0": "Success",
					"1": "API/runtime error",
					"2": "Usage error",
					"3": "Auth/config error",
				},
				AgentNotes: []string{
					"Flags under root_flags (e.g. --version) are accepted only on the bare 'uploadcare' command, not on subcommands — do not prepend them to subcommand invocations.",
					"The --json flag requires a value: --json all (every field) or --json field1,field2 (specific fields).",
					"The --jq flag implies --json. You do not need to pass both --json and --jq.",
					"All timestamps are in RFC 3339 / UTC format.",
					"For batch operations (file store, file delete), exit code 1 means partial success — check the 'problems' field in JSON output.",
					"When piping between commands, use --json uuid or --jq '.uuid' to emit just the UUID for --from-stdin consumption.",
					"For file download, use --output-dir for multiple UUIDs or --from-stdin; --output - streams bytes to stdout and cannot be combined with --json.",
					"File search requires at least one query or filter. Search results can lag recent file, metadata, and tag changes. The API serves at most the first 1000 matches of a search.",
					"Tag flags are repeatable. Tag update applies all --delete values before all --add values. Tag mutations with --dry-run add \"status\": \"would change\" to their JSON output.",
					"For 'project usage': --to must be strictly before today in UTC. Using today's date or a future date will return a validation error.",
				},
				URLAPI: buildURLAPISchema(),
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(schema)
		},
	}
}

// rootOnlyFlags returns flags accepted only on the bare "uploadcare" command
// (currently --version). Unlike global flags, these are NOT inherited by
// subcommands, so consumers must not prepend them to subcommand invocations.
// The auto-generated --help flag is omitted.
func rootOnlyFlags(root *cobra.Command) []flagSchema {
	var flags []flagSchema
	root.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		flags = append(flags, flagToSchema(f))
	})
	return flags
}

func collectFlags(fs *pflag.FlagSet) []flagSchema {
	var flags []flagSchema
	fs.VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagToSchema(f))
	})
	return flags
}

func flagToSchema(f *pflag.Flag) flagSchema {
	return flagSchema{
		Name:        f.Name,
		Type:        f.Value.Type(),
		Default:     f.DefValue,
		Description: f.Usage,
	}
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
			Path:       path,
			Short:      sub.Short,
			Long:       sub.Long,
			Args:       extractArgs(sub),
			Flags:      collectFlags(sub.LocalFlags()),
			JSONFields: jsonFieldsForCommand(path),
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

// extractArgs derives argument bounds by probing the command's Args
// validator with growing argument lists — the validator is what the CLI
// actually enforces, while the Use string is display-only and ambiguous
// (e.g. "<uuid> <tag>..." with MinimumNArgs(2)).
func extractArgs(cmd *cobra.Command) argsSchema {
	validator := cmd.Args
	if validator == nil {
		return argsSchema{Min: 0, Max: 0}
	}

	const probeMax = 8
	minArgs, maxArgs := -1, -1
	for n := 0; n <= probeMax; n++ {
		if validator(cmd, make([]string, n)) != nil {
			continue
		}
		if minArgs == -1 {
			minArgs = n
		}
		maxArgs = n
	}
	if minArgs == -1 {
		return argsSchema{Min: 0, Max: 0}
	}
	if validator(cmd, make([]string, probeMax*8)) == nil {
		maxArgs = -1 // unlimited
	}
	return argsSchema{Min: minArgs, Max: maxArgs}
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

// jsonFieldsForCommand returns the known JSON field names for a command.
// These correspond to the struct JSON tags used when --json output is active.
func jsonFieldsForCommand(path string) []string {
	fileFields := []string{"uuid", "size", "filename", "mime_type", "is_image", "is_stored", "is_ready", "datetime_uploaded", "datetime_stored", "datetime_removed", "url", "original_file_url", "metadata", "tags", "appdata"}

	switch path {
	case "file info", "file upload", "file upload-from-url", "file local-copy":
		return fileFields
	case "file list":
		return fileFields
	case "file search":
		return append(fileFields, "highlight")
	case "tag list":
		return []string{"tags"}
	case "tag replace", "tag update", "tag clear":
		return []string{"tags", "added", "deleted", "status"}
	case "file store", "file delete":
		return []string{"results", "problems"}
	case "file remote-copy":
		return []string{"type", "result", "already_exists"}
	case "file download":
		return []string{"uuid", "path", "size", "source_url", "status"}
	case "group info", "group create", "group list":
		return []string{"id", "datetime_created", "datetime_stored", "files_count", "cdn_url", "url", "files"}
	case "webhook list", "webhook create", "webhook update":
		return []string{"id", "target_url", "event", "is_active", "signing_secret", "created", "updated"}
	case "addon execute", "addon status":
		return []string{"status", "result"}
	case "convert document", "convert video":
		return []string{"token", "uuid", "status"}
	case "version":
		return []string{"version", "commit", "date", "go_version", "os", "arch"}
	default:
		return nil
	}
}
