package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/config"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func newProjectCmd(
	projectSvc service.ProjectService,
	mgmtSvc service.ProjectManagementService,
	secretSvc service.SecretService,
	usageSvc service.UsageService,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long: `Manage Uploadcare projects and their settings.

"project info" (no args) uses the REST API with --public-key/--secret-key.
All other commands use the Project API with --project-api-token.

Where a <project> argument is required, you can pass either a project
name from your config file or a pub_key directly.

Subcommands: info, list, create, update, delete, use, secret, usage.`,
	}

	cmd.AddCommand(newProjectInfoCmd(projectSvc, mgmtSvc))
	cmd.AddCommand(newProjectListCmd(mgmtSvc))
	cmd.AddCommand(newProjectCreateCmd(mgmtSvc, secretSvc))
	cmd.AddCommand(newProjectUpdateCmd(mgmtSvc))
	cmd.AddCommand(newProjectDeleteCmd(mgmtSvc))
	cmd.AddCommand(newProjectUseCmd(mgmtSvc, secretSvc))
	cmd.AddCommand(newProjectSecretCmd(secretSvc))
	cmd.AddCommand(newProjectUsageCmd(usageSvc))

	return cmd
}

func newProjectInfoCmd(projectSvc service.ProjectService, mgmtSvc service.ProjectManagementService) *cobra.Command {
	return &cobra.Command{
		Use:   "info [project]",
		Short: "Get project details",
		Long: `Get project details.

Without arguments, uses the REST API (requires --public-key/--secret-key)
and returns info about the currently configured project.

With a <project> argument (name or pub_key), uses the Project API
(requires --project-api-token) and returns detailed project settings.

JSON fields (REST): name, pub_key, collaborators, autostore_enabled.
JSON fields (Project API): pub_key, name, is_blocked, is_shared_project,
filesize_limit, autostore_enabled.`,
		Example: `  # Info about the current project (REST API)
  uploadcare project info

  # Info about a specific project by pub_key
  uploadcare project info abc123 --project-api-token <token>

  # Info about a project by config name
  uploadcare project info "My App" --project-api-token <token>`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if len(args) == 0 {
				// REST API mode
				svc := projectSvc
				if svc == nil {
					var err error
					svc, err = projectServiceFromCmd(cmd)
					if err != nil {
						return err
					}
				}

				p, err := svc.Info(cmd.Context())
				if err != nil {
					return err
				}

				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), p)
				}

				table := &output.TableData{}
				table.AddRow("Name:", p.Name)
				table.AddRow("Public Key:", p.PubKey)
				table.AddRow("Autostore:", strconv.FormatBool(p.AutostoreEnabled))
				if len(p.Collaborators) > 0 {
					for i, c := range p.Collaborators {
						label := "Collaborator:"
						if i > 0 {
							label = ""
						}
						table.AddRow(label, fmt.Sprintf("%s <%s>", c.Name, c.Email))
					}
				}
				return formatter.Format(cmd.OutOrStdout(), table)
			}

			// Project API mode
			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			pubKey := resolveProjectPubKey(cmd, args[0])
			p, err := svc.Get(cmd.Context(), pubKey)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), p)
			}

			table := &output.TableData{}
			table.AddRow("Name:", p.Name)
			table.AddRow("Public Key:", p.PubKey)
			if p.AutostoreEnabled != nil {
				table.AddRow("Autostore:", strconv.FormatBool(*p.AutostoreEnabled))
			}
			if p.FilesizeLimit != nil {
				table.AddRow("Filesize Limit:", strconv.FormatInt(*p.FilesizeLimit, 10))
			}
			if p.IsBlocked != nil {
				table.AddRow("Blocked:", strconv.FormatBool(*p.IsBlocked))
			}
			table.AddRow("Shared:", strconv.FormatBool(p.IsSharedProject))
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newProjectListCmd(mgmtSvc service.ProjectManagementService) *cobra.Command {
	var pageAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long: `List all projects accessible with the current bearer token.

Requires --project-api-token.

JSON fields: pub_key, name, is_blocked, is_shared_project,
filesize_limit, autostore_enabled.`,
		Example: `  # List projects as a table
  uploadcare project list --project-api-token <token>

  # List as JSON
  uploadcare project list --json all --project-api-token <token>

  # Stream all pages as NDJSON
  uploadcare project list --page-all --json all --project-api-token <token>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			result, err := svc.List(cmd.Context(), service.ProjectListOptions{PageAll: pageAll})
			if err != nil {
				return err
			}

			if pageAll && opts.JSON {
				for _, p := range result.Projects {
					if err := output.NDJSONLine(cmd.OutOrStdout(), p, opts.Fields, opts.JQ); err != nil {
						return err
					}
				}
				return nil
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result.Projects)
			}

			table := output.NewTableData("PUB_KEY", "NAME", "SHARED", "BLOCKED")
			for _, p := range result.Projects {
				blocked := ""
				if p.IsBlocked != nil && *p.IsBlocked {
					blocked = "true"
				}
				table.AddRow(
					p.PubKey,
					p.Name,
					strconv.FormatBool(p.IsSharedProject),
					blocked,
				)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	cmd.Flags().BoolVar(&pageAll, "page-all", false, "Fetch all pages as NDJSON stream")

	return cmd
}

func newProjectCreateCmd(mgmtSvc service.ProjectManagementService, secretSvc service.SecretService) *cobra.Command {
	var (
		noSave        bool
		useAsDefault  bool
		filesizeLimit int64
		autostore     string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long: `Create a new project and optionally save it to the config file.

By default, also creates an API secret and saves the project to the
config file (~/.uploadcare/config.yaml). The project name is used as
the key in the config "projects:" map.

Use --no-save to skip saving credentials to the config.
Use --use to set the new project as the default.

Requires --project-api-token.`,
		Example: `  # Create a project (saves to config by default)
  uploadcare project create "My App" --project-api-token <token>

  # Create and set as default
  uploadcare project create "My App" --use --project-api-token <token>

  # Create without saving to config
  uploadcare project create "My App" --no-save --project-api-token <token>

  # Dry run
  uploadcare project create "My App" --dry-run --project-api-token <token>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if noSave && useAsDefault {
				return ExitErrorf(2, "--no-save and --use are mutually exclusive: --use requires saving to config")
			}

			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			secSvc := secretSvc
			if secSvc == nil {
				var err error
				secSvc, err = secretServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			params := service.ProjectCreateParams{Name: name}
			if cmd.Flags().Changed("filesize-limit") {
				params.FilesizeLimit = &filesizeLimit
			}
			if cmd.Flags().Changed("autostore") {
				b, err := strconv.ParseBool(autostore)
				if err != nil {
					return ExitErrorf(2, "invalid --autostore value: %q", autostore)
				}
				params.AutostoreEnabled = &b
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"name":   name,
						"status": "would create",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would create project %q\n", name)
				return nil
			}

			// Create the project.
			p, err := svc.Create(cmd.Context(), params)
			if err != nil {
				return err
			}

			// Auto-create a secret.
			secret, err := secSvc.Create(cmd.Context(), p.PubKey)
			if err != nil {
				// Project was created but secret creation failed.
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: project created but secret creation failed: %v\n", err)
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), p)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created project %q (pub_key: %s)\n", p.Name, p.PubKey)
				return nil
			}

			// Save to config unless --no-save.
			saved := false
			if !noSave {
				if err := config.SaveProjectEntry(name, p.PubKey, secret.Secret); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to save to config: %v\n", err)
				} else {
					saved = true
				}
			}

			// Set as default only if save succeeded.
			madeDefault := false
			if useAsDefault && saved {
				if err := config.SetDefaultProject(name); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to set default project: %v\n", err)
				} else {
					madeDefault = true
				}
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
					"pub_key":    p.PubKey,
					"name":       p.Name,
					"secret_id":  secret.ID,
					"secret_key": secret.Secret,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created project %q (pub_key: %s)\n", p.Name, p.PubKey)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created API secret (shown once): %s\n", secret.Secret)
			if saved {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved project %q to config\n", name)
				if madeDefault {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %q as default project\n", name)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nUse with: uploadcare file list\n")
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nUse with: uploadcare --project %q file list\n", name)
				}
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&noSave, "no-save", false, "Do not save project credentials to config file")
	f.BoolVar(&useAsDefault, "use", false, "Set as the default project in config")
	f.Int64Var(&filesizeLimit, "filesize-limit", 0, "Max file size in bytes")
	f.StringVar(&autostore, "autostore", "", "Auto-store uploads (true/false)")
	f.BoolVar(&dryRun, "dry-run", false, "Validate inputs, do not create")

	return cmd
}

func newProjectUpdateCmd(mgmtSvc service.ProjectManagementService) *cobra.Command {
	var (
		name          string
		filesizeLimit int64
		autostore     string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "update <project>",
		Short: "Update project settings",
		Long: `Update project settings by name or pub_key.

Only the flags you provide are changed — omitted flags leave the
current value unchanged.

Requires --project-api-token.

JSON fields: pub_key, name, is_blocked, filesize_limit, autostore_enabled.`,
		Example: `  # Rename a project
  uploadcare project update "My App" --name "Production"

  # Set filesize limit
  uploadcare project update abc123 --filesize-limit 10485760

  # Dry run
  uploadcare project update "My App" --name "New Name" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])

			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			params := service.ProjectUpdateParams{}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("filesize-limit") {
				params.FilesizeLimit = &filesizeLimit
			}
			if cmd.Flags().Changed("autostore") {
				b, err := strconv.ParseBool(autostore)
				if err != nil {
					return ExitErrorf(2, "invalid --autostore value: %q", autostore)
				}
				params.AutostoreEnabled = &b
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"pub_key": pubKey,
						"status":  "would update",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would update project %s\n", pubKey)
				return nil
			}

			p, err := svc.Update(cmd.Context(), pubKey, params)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), p)
			}

			table := &output.TableData{}
			table.AddRow("Name:", p.Name)
			table.AddRow("Public Key:", p.PubKey)
			if p.AutostoreEnabled != nil {
				table.AddRow("Autostore:", strconv.FormatBool(*p.AutostoreEnabled))
			}
			if p.FilesizeLimit != nil {
				table.AddRow("Filesize Limit:", strconv.FormatInt(*p.FilesizeLimit, 10))
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&name, "name", "", "New project name")
	f.Int64Var(&filesizeLimit, "filesize-limit", 0, "Max file size in bytes")
	f.StringVar(&autostore, "autostore", "", "Auto-store uploads (true/false)")
	f.BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")

	return cmd
}

func newProjectDeleteCmd(mgmtSvc service.ProjectManagementService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <project>",
		Short: "Delete a project",
		Long: `Soft-delete a project by name or pub_key.

Only the project owner can delete. Also removes the project from the
config file if present.

Requires --project-api-token.`,
		Example: `  # Delete by pub_key
  uploadcare project delete abc123

  # Delete by config name
  uploadcare project delete "My App"

  # Dry run
  uploadcare project delete "My App" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])

			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"pub_key": pubKey,
						"status":  "would delete",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would delete project %s\n", pubKey)
				return nil
			}

			if err := svc.Delete(cmd.Context(), pubKey); err != nil {
				return err
			}

			// Best-effort removal from config by pub_key.
			// This works whether the user passed a name or a raw pub_key.
			_ = config.RemoveProjectByPubKey(pubKey)

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"pub_key": pubKey,
					"status":  "deleted",
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted project %s\n", pubKey)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Verify without deleting")

	return cmd
}

func newProjectUseCmd(mgmtSvc service.ProjectManagementService, secretSvc service.SecretService) *cobra.Command {
	var (
		secretKey    string
		createSecret bool
		useAsDefault bool
	)

	cmd := &cobra.Command{
		Use:   "use <project>",
		Short: "Switch active project",
		Long: `Save a project's credentials to the config file.

Fetches the project name from the API and uses it as the config key.
One of --secret-key or --create-secret is required.

Requires --project-api-token.`,
		Example: `  # Switch to a project with an existing secret
  uploadcare project use abc123 --secret-key "my-secret" --use

  # Switch and create a fresh secret
  uploadcare project use abc123 --create-secret

  # Use project by config name
  uploadcare project use "Staging" --create-secret --use`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("secret-key") && !createSecret {
				return ExitErrorf(2, "one of --secret-key or --create-secret is required")
			}
			if cmd.Flags().Changed("secret-key") && secretKey == "" {
				return ExitErrorf(2, "--secret-key must not be empty")
			}
			if cmd.Flags().Changed("secret-key") && createSecret {
				return ExitErrorf(2, "--secret-key and --create-secret are mutually exclusive")
			}

			pubKey := resolveProjectPubKey(cmd, args[0])

			svc := mgmtSvc
			if svc == nil {
				var err error
				svc, err = projectMgmtServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			// Fetch project to get its name.
			p, err := svc.Get(cmd.Context(), pubKey)
			if err != nil {
				return err
			}

			// Resolve secret key.
			sk := secretKey
			if createSecret {
				secSvc := secretSvc
				if secSvc == nil {
					secSvc, err = secretServiceFromCmd(cmd)
					if err != nil {
						return err
					}
				}
				result, err := secSvc.Create(cmd.Context(), pubKey)
				if err != nil {
					return err
				}
				sk = result.Secret
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Created API secret (shown once): %s\n", result.Secret)
			}

			// Save to config.
			if err := config.SaveProjectEntry(p.Name, pubKey, sk); err != nil {
				return fmt.Errorf("saving to config: %w", err)
			}

			madeDefault := false
			if useAsDefault {
				if err := config.SetDefaultProject(p.Name); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to set default project: %v\n", err)
				} else {
					madeDefault = true
				}
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
					"pub_key": pubKey,
					"name":    p.Name,
					"status":  "saved",
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved project %q to config\n", p.Name)
			if madeDefault {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %q as default project\n", p.Name)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&secretKey, "secret-key", "", "Use this existing secret key")
	f.BoolVar(&createSecret, "create-secret", false, "Create a new API secret for this project")
	f.BoolVar(&useAsDefault, "use", false, "Set as the default project in config")

	return cmd
}
