package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/humzakhan/recap/internal/aiclient"
	"github.com/humzakhan/recap/internal/config"
	"github.com/humzakhan/recap/internal/extractor"
	"github.com/humzakhan/recap/internal/fetcher"
	"github.com/humzakhan/recap/internal/log"
	"github.com/humzakhan/recap/internal/server"
	"github.com/humzakhan/recap/internal/template"
)

// fetcherAdapter wraps the fetcher.Fetch function to satisfy the extractor.Fetcher interface.
type fetcherAdapter struct{}

func (f fetcherAdapter) Fetch(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	return fetcher.Fetch(ctx, rawURL)
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var verbose bool

	root := &cobra.Command{
		Use:   "recap",
		Short: "Extract structured data from any URL",
		Long:  "recap is a CLI + web app that extracts structured, user-defined fields from any URL using an LLM.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetVerbose(verbose)
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug output")

	root.AddCommand(runCmd())
	root.AddCommand(templateCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(configCmd())

	return root
}

// buildAIClient creates an AIClient from the loaded config.
func buildAIClient(cfg *config.Config) (aiclient.AIClient, error) {
	// Determine provider from model.
	provider := ""
	if info, ok := aiclient.LookupModel(cfg.Model); ok {
		provider = info.Provider
	} else {
		provider = aiclient.InferProvider(cfg.Model)
	}
	if provider == "" {
		return nil, fmt.Errorf("cannot determine provider for model %q; set the model to a known model or configure api_keys in config", cfg.Model)
	}

	apiKey := cfg.ProviderAPIKey(provider)
	return aiclient.New(aiclient.ProviderConfig{
		Provider: provider,
		APIKey:   apiKey,
		Model:    cfg.Model,
	})
}

// ---------------------------------------------------------------------------
// recap run
// ---------------------------------------------------------------------------

func runCmd() *cobra.Command {
	var (
		fieldsFlag   string
		templateFlag string
		formatFlag   string
		showRaw      bool
		outputFlag   string
	)

	cmd := &cobra.Command{
		Use:   "run <url>",
		Short: "Extract structured data from a URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			log.Debug("run: starting extraction for %q", url)

			// Parse fields from flag.
			parsed, err := ParseFields(fieldsFlag)
			if err != nil {
				return fmt.Errorf("invalid --fields syntax: %w\n  Expected format: \"title,author:text,tags:list,sentiment:enum[pos,neg]\"", err)
			}

			// Load template if specified.
			var templateFields []extractor.Field
			if templateFlag != "" {
				log.Debug("run: loading template %q", templateFlag)
				tmpl, err := template.LoadTemplate(templateFlag)
				if err != nil {
					return fmt.Errorf("template %q not found: %w\n  Run 'recap template list' to see available templates", templateFlag, err)
				}
				templateFields = tmpl.Fields
				log.Debug("run: template %q loaded with %d fields", templateFlag, len(templateFields))
			}

			// Merge template fields with parsed fields.
			fields := extractor.MergeFields(templateFields, parsed)
			if len(fields) == 0 {
				return fmt.Errorf("no fields specified\n  Use --fields \"title,author,tags:list\" or --template news-article")
			}

			log.Debug("run: %d fields to extract", len(fields))
			for _, f := range fields {
				log.Debug("run:   - %s (%s)", f.Label, f.Type)
			}

			// Load config.
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			log.Debug("run: config loaded (model=%s, max_tokens=%d)", cfg.Model, cfg.MaxTokens)

			// Build AI client.
			client, err := buildAIClient(cfg)
			if err != nil {
				return err
			}

			// Determine output format.
			format := formatFlag
			if format == "" {
				format = cfg.DefaultFormat
			}

			// Create extractor and run.
			log.Debug("run: starting extraction (format=%s)", format)
			ext := extractor.NewExtractor(fetcherAdapter{}, client, cfg.MaxTokens)
			result, err := ext.Extract(context.Background(), extractor.ExtractionRequest{
				URL:    url,
				Fields: fields,
				Format: format,
			})
			if err != nil {
				return err
			}
			log.Debug("run: extraction complete, formatting as %s", format)

			// Format output.
			var output string
			switch format {
			case "markdown":
				output = formatMarkdown(result, fields)
			case "csv":
				output = formatCSV(result, fields)
			default:
				output, err = formatJSON(result)
				if err != nil {
					return err
				}
			}

			// Show raw content if requested.
			if showRaw {
				output += "\n\n--- Raw Content ---\n\n" + result.RawContent
			}

			// Write to file or stdout.
			if outputFlag != "" {
				if err := os.WriteFile(outputFlag, []byte(output), 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Output written to %s\n", outputFlag)
			} else {
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&fieldsFlag, "fields", "f", "", `Comma-separated field definitions (e.g., "title,author:text,tags:list")`)
	cmd.Flags().StringVarP(&templateFlag, "template", "t", "", "Template name or path to a .json template file")
	cmd.Flags().StringVar(&formatFlag, "format", "", "Output format: json (default), markdown, csv")
	cmd.Flags().BoolVar(&showRaw, "show-raw", false, "Also print the raw fetched content")
	cmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to a file instead of stdout")

	return cmd
}

// ---------------------------------------------------------------------------
// recap template
// ---------------------------------------------------------------------------

func templateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage extraction templates",
	}

	cmd.AddCommand(templateListCmd())
	cmd.AddCommand(templateShowCmd())
	cmd.AddCommand(templateSaveCmd())
	cmd.AddCommand(templateDeleteCmd())
	cmd.AddCommand(templateImportCmd())
	cmd.AddCommand(templateExportCmd())

	return cmd
}

func templateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			templates, err := template.ListTemplates()
			if err != nil {
				return err
			}

			if len(templates) == 0 {
				fmt.Println("No templates found.")
				return nil
			}

			for _, t := range templates {
				desc := t.Description
				if desc == "" {
					desc = "(no description)"
				}
				fmt.Printf("  %-20s %s\n", t.Name, desc)
			}
			return nil
		},
	}
}

func templateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := template.LoadTemplate(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Name:        %s\n", tmpl.Name)
			if tmpl.Description != "" {
				fmt.Printf("Description: %s\n", tmpl.Description)
			}
			fmt.Printf("Fields:\n")
			for _, f := range tmpl.Fields {
				line := fmt.Sprintf("  - %s (%s)", f.Label, f.Type)
				if len(f.Options) > 0 {
					line += fmt.Sprintf(" [%s]", joinOptions(f.Options))
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

func templateSaveCmd() *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save a new user template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			fields, err := ParseFields(fieldsFlag)
			if err != nil {
				return fmt.Errorf("parsing fields: %w", err)
			}
			if len(fields) == 0 {
				return fmt.Errorf("no fields specified; use --fields")
			}

			tmpl := &extractor.Template{
				Name:   name,
				Fields: fields,
			}
			if err := template.SaveUserTemplate(tmpl); err != nil {
				return err
			}
			fmt.Printf("Template %q saved.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&fieldsFlag, "fields", "f", "", `Comma-separated field definitions`)
	_ = cmd.MarkFlagRequired("fields")

	return cmd
}

func templateDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a user template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := template.DeleteUserTemplate(args[0]); err != nil {
				return err
			}
			fmt.Printf("Template %q deleted.\n", args[0])
			return nil
		},
	}
}

func templateImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file.json>",
		Short: "Import a template from a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := template.ImportTemplate(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Template %q imported.\n", tmpl.Name)
			return nil
		},
	}
}

func templateExportCmd() *cobra.Command {
	var outputFlag string

	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a template to a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := template.ExportTemplate(args[0], outputFlag); err != nil {
				return err
			}
			target := outputFlag
			if target == "" {
				target = args[0] + ".json"
			}
			fmt.Printf("Template %q exported to %s\n", args[0], target)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path")

	return cmd
}

// ---------------------------------------------------------------------------
// recap serve
// ---------------------------------------------------------------------------

func serveCmd() *cobra.Command {
	var (
		port int
		host string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Override config with CLI flags if provided.
			if cmd.Flags().Changed("port") {
				cfg.Server.Port = port
			}
			if cmd.Flags().Changed("host") {
				cfg.Server.Host = host
			}

			// Validate that at least one API key is configured.
			client, err := buildAIClient(cfg)
			if err != nil {
				return err
			}

			fmt.Printf("Starting server on %s:%d (model: %s)\n", cfg.Server.Host, cfg.Server.Port, cfg.Model)
			srv := server.New(cfg, client)
			return srv.Run()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host to bind to")

	return cmd
}

// ---------------------------------------------------------------------------
// recap config
// ---------------------------------------------------------------------------

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and manage configuration",
	}

	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configModelsCmd())

	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration (API keys redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			redacted := config.Redact(cfg)
			data, err := yaml.Marshal(redacted)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("config set not yet implemented")
			return nil
		},
	}
}

func configModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			grouped := aiclient.ModelsByProvider()

			// Sort provider names for consistent output.
			providers := make([]string, 0, len(grouped))
			for p := range grouped {
				providers = append(providers, p)
			}
			sort.Strings(providers)

			for _, provider := range providers {
				models := grouped[provider]
				hasKey := cfg.ProviderAPIKey(provider) != ""

				keyStatus := "(no API key)"
				if hasKey {
					keyStatus = "(configured)"
				}
				fmt.Printf("%s %s\n", strings.ToUpper(provider), keyStatus)

				for _, m := range models {
					marker := "  "
					if m.ID == cfg.Model {
						marker = "* "
					}
					fmt.Printf("  %s%-35s %s\n", marker, m.ID, m.Name)
				}
				fmt.Println()
			}

			fmt.Printf("Current model: %s\n", cfg.Model)
			fmt.Println("Change with: RECAP_MODEL=<model-id> or set 'model' in ~/.recap/config.yaml")

			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func joinOptions(opts []string) string {
	result := ""
	for i, o := range opts {
		if i > 0 {
			result += ", "
		}
		result += o
	}
	return result
}
