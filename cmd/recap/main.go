package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/humzakhan/recap/internal/config"
	"github.com/humzakhan/recap/internal/extractor"
	"github.com/humzakhan/recap/internal/fetcher"
	"github.com/humzakhan/recap/internal/template"
)

// fetcherAdapter wraps the fetcher.Fetch function to satisfy the extractor.Fetcher interface.
type fetcherAdapter struct{}

func (f fetcherAdapter) Fetch(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	return fetcher.Fetch(ctx, rawURL)
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "recap",
		Short: "Extract structured data from any URL",
		Long:  "recap is a CLI + web app that extracts structured, user-defined fields from any URL using an LLM.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(runCmd())
	root.AddCommand(templateCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(configCmd())

	return root
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

			// Parse fields from flag.
			parsed, err := ParseFields(fieldsFlag)
			if err != nil {
				return fmt.Errorf("parsing fields: %w", err)
			}

			// Load template if specified.
			var templateFields []extractor.Field
			if templateFlag != "" {
				tmpl, err := template.LoadTemplate(templateFlag)
				if err != nil {
					return fmt.Errorf("loading template: %w", err)
				}
				templateFields = tmpl.Fields
			}

			// Merge template fields with parsed fields.
			fields := extractor.MergeFields(templateFields, parsed)
			if len(fields) == 0 {
				return fmt.Errorf("no fields specified; use --fields or --template")
			}

			// Load config.
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if cfg.AnthropicAPIKey == "" {
				return fmt.Errorf("no API key configured; set RECAP_ANTHROPIC_KEY or configure via ~/.recap/config.yaml")
			}

			// Determine output format.
			format := formatFlag
			if format == "" {
				format = cfg.DefaultFormat
			}

			// Create extractor and run.
			ext := extractor.NewExtractor(fetcherAdapter{}, cfg.AnthropicAPIKey, cfg.Model, cfg.MaxTokens)
			result, err := ext.Extract(context.Background(), extractor.ExtractionRequest{
				URL:    url,
				Fields: fields,
				Format: format,
			})
			if err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}

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
			fmt.Printf("Server not yet implemented, will listen on %s:%d\n", host, port)
			return nil
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

	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration (API key redacted)",
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
