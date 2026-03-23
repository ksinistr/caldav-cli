package cmd

import (
	"context"
	"fmt"

	"github.com/ksinistr/caldav-cli/internal/caldav"
	"github.com/ksinistr/caldav-cli/internal/config"
	"github.com/ksinistr/caldav-cli/internal/errors"
	"github.com/ksinistr/caldav-cli/internal/out"
	"github.com/urfave/cli/v3"
)

// CalendarsListAction implements the `caldav calendars list` command.
func CalendarsListAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	configPath := cmd.String("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create CalDAV client
	client, err := caldav.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	// Discover calendars
	result, err := caldav.DiscoverCalendars(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to discover calendars: %w", err)
	}

	// Render output
	formatResolver := out.NewFormatResolver(cmd.String("format"))
	format := formatResolver.Resolve()
	return out.Write(ctx, format, result)
}

// RequireFormatFlag ensures format is explicitly set for agent-friendly output.
// This is used to guide agents to always use --format for consistent output.
func RequireFormatFlag(cmd *cli.Command) error {
	if cmd.String("format") == "" {
		return out.WriteError(context.Background(), out.FormatText, fmt.Errorf("--format flag is required (use: json, toon, or text)"))
	}
	return nil
}

// GetConfig loads and validates config from the CLI context.
func GetConfig(cmd *cli.Command) (*config.Config, error) {
	configPath := cmd.String("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// GetClient creates a CalDAV client from the CLI context.
func GetClient(cfg *config.Config) (*caldav.Client, error) {
	client, err := caldav.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}
	return client, nil
}

// WriteOutput writes data to stdout using the format from the CLI context.
func WriteOutput(ctx context.Context, cmd *cli.Command, data any) error {
	formatResolver := out.NewFormatResolver(cmd.String("format"))
	format := formatResolver.Resolve()
	return out.Write(ctx, format, data)
}

// WriteError writes an error to stderr using the format from the CLI context.
// It returns a FormattedError sentinel to prevent double-printing in cli.Run.
func WriteError(ctx context.Context, cmd *cli.Command, err error) error {
	formatResolver := out.NewFormatResolver(cmd.String("format"))
	format := formatResolver.Resolve()
	// Write the formatted error to stderr
	_ = out.WriteError(ctx, format, err)
	// Return a sentinel error so cli.Run knows the error was already formatted
	return errors.NewFormattedError(err)
}
