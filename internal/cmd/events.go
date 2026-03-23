package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ksinistr/caldav-cli/internal/caldav"
	"github.com/urfave/cli/v3"
)

// EventsListAction implements the `caldav events list` command.
func EventsListAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	// Create CalDAV client
	client, err := GetClient(cfg)
	if err != nil {
		return err
	}

	// Parse command line arguments
	calendarID := cmd.String("calendar-id")
	fromStr := cmd.String("from")
	toStr := cmd.String("to")

	// Parse date range
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("invalid --from date: must be ISO 8601 format (e.g., 2026-03-24T00:00:00Z): %w", err))
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("invalid --to date: must be ISO 8601 format (e.g., 2026-03-31T00:00:00Z): %w", err))
	}

	// Validate date range
	if to.Before(from) {
		return WriteError(ctx, cmd, fmt.Errorf("invalid date range: --to must be after --from"))
	}

	// List events
	result, err := caldav.ListEvents(ctx, client, calendarID, from, to)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("failed to list events: %w", err))
	}

	// Render output
	return WriteOutput(ctx, cmd, result)
}

// EventsGetAction implements the `caldav events get` command.
func EventsGetAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	// Create CalDAV client
	client, err := GetClient(cfg)
	if err != nil {
		return err
	}

	// Parse command line arguments
	calendarID := cmd.String("calendar-id")
	eventID := cmd.String("event-id")

	// Fetch the event
	result, err := caldav.GetEvent(ctx, client, calendarID, eventID)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("failed to get event: %w", err))
	}

	// Render output
	return WriteOutput(ctx, cmd, result)
}

// EventsCreateAction implements the `caldav events create` command.
func EventsCreateAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	// Create CalDAV client
	client, err := GetClient(cfg)
	if err != nil {
		return err
	}

	// Parse command line arguments
	calendarID := cmd.String("calendar-id")
	title := cmd.String("title")
	description := cmd.String("description")
	startStr := cmd.String("start")
	endStr := cmd.String("end")
	dateStr := cmd.String("date")

	// Build the create input
	input := caldav.CreateEventInput{
		CalendarID:  calendarID,
		Title:       title,
		Description: description,
	}

	// Parse timed event flags
	if startStr != "" || endStr != "" {
		if startStr == "" {
			return WriteError(ctx, cmd, fmt.Errorf("--end provided without --start"))
		}
		if endStr == "" {
			return WriteError(ctx, cmd, fmt.Errorf("--start provided without --end"))
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return WriteError(ctx, cmd, fmt.Errorf("invalid --start date: must be ISO 8601 format (e.g., 2026-03-25T09:00:00Z): %w", err))
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return WriteError(ctx, cmd, fmt.Errorf("invalid --end date: must be ISO 8601 format (e.g., 2026-03-25T09:30:00Z): %w", err))
		}

		if end.Before(start) {
			return WriteError(ctx, cmd, fmt.Errorf("invalid date range: --end must be after --start"))
		}

		input.Start = &start
		input.End = &end
	}

	// Parse all-day event flag
	if dateStr != "" {
		if startStr != "" || endStr != "" {
			return WriteError(ctx, cmd, fmt.Errorf("cannot use --date with --start/--end flags"))
		}

		// Parse date in YYYY-MM-DD format
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return WriteError(ctx, cmd, fmt.Errorf("invalid --date: must be YYYY-MM-DD format (e.g., 2026-03-27): %w", err))
		}

		input.Date = &date
	}

	// Create the event
	result, err := caldav.CreateEvent(ctx, client, input)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("failed to create event: %w", err))
	}

	// Render output
	return WriteOutput(ctx, cmd, result)
}

// EventsUpdateAction implements the `caldav events update` command.
func EventsUpdateAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	// Create CalDAV client
	client, err := GetClient(cfg)
	if err != nil {
		return err
	}

	// Parse command line arguments
	calendarID := cmd.String("calendar-id")
	eventID := cmd.String("event-id")

	// Build the update input - only set fields that were provided
	input := caldav.UpdateEventInput{
		CalendarID: calendarID,
		EventID:    eventID,
	}

	// Parse optional flags
	if cmd.IsSet("title") {
		title := cmd.String("title")
		input.Title = &title
	}

	if cmd.IsSet("description") {
		description := cmd.String("description")
		input.Description = &description
	}

	// Parse timed event flags
	if cmd.IsSet("start") || cmd.IsSet("end") {
		if cmd.IsSet("start") && cmd.IsSet("end") {
			startStr := cmd.String("start")
			endStr := cmd.String("end")

			start, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				return WriteError(ctx, cmd, fmt.Errorf("invalid --start date: must be ISO 8601 format (e.g., 2026-03-25T09:00:00Z): %w", err))
			}

			end, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				return WriteError(ctx, cmd, fmt.Errorf("invalid --end date: must be ISO 8601 format (e.g., 2026-03-25T09:30:00Z): %w", err))
			}

			if end.Before(start) {
				return WriteError(ctx, cmd, fmt.Errorf("invalid date range: --end must be after --start"))
			}

			input.Start = &start
			input.End = &end
		} else if cmd.IsSet("start") {
			return WriteError(ctx, cmd, fmt.Errorf("--end required when --start is provided"))
		} else {
			return WriteError(ctx, cmd, fmt.Errorf("--start required when --end is provided"))
		}
	}

	// Parse all-day event flag
	if cmd.IsSet("date") {
		if cmd.IsSet("start") || cmd.IsSet("end") {
			return WriteError(ctx, cmd, fmt.Errorf("cannot use --date with --start/--end flags"))
		}

		dateStr := cmd.String("date")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return WriteError(ctx, cmd, fmt.Errorf("invalid --date: must be YYYY-MM-DD format (e.g., 2026-03-27): %w", err))
		}

		input.Date = &date
	}

	// Update the event
	result, err := caldav.UpdateEvent(ctx, client, input)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("failed to update event: %w", err))
	}

	// Render output
	return WriteOutput(ctx, cmd, result)
}

// EventsDeleteAction implements the `caldav events delete` command.
func EventsDeleteAction(ctx context.Context, cmd *cli.Command) error {
	// Load config
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	// Create CalDAV client
	client, err := GetClient(cfg)
	if err != nil {
		return err
	}

	// Parse command line arguments
	calendarID := cmd.String("calendar-id")
	eventID := cmd.String("event-id")

	// Delete the event
	result, err := caldav.DeleteEvent(ctx, client, calendarID, eventID)
	if err != nil {
		return WriteError(ctx, cmd, fmt.Errorf("failed to delete event: %w", err))
	}

	// Render output
	return WriteOutput(ctx, cmd, result)
}
