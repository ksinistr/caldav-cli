package cli

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/ksinistr/caldav-cli/internal/cmd"
	internalerrors "github.com/ksinistr/caldav-cli/internal/errors"
	"github.com/ksinistr/caldav-cli/internal/out"
	"github.com/urfave/cli/v3"
)

var errCommandRequired = stderrors.New("command required")

// New creates a new CLI application instance.
func New() *cli.Command {
	return &cli.Command{
		Name:  "caldav",
		Usage: "CalDAV CLI for Baikal calendar server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to config file (default: ~/.config/caldav-cli/config.toml)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: json|toon|text (default: toon for non-TTY, text for TTY)",
				Value: "",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "calendars",
				Usage: "Manage calendars",
				Commands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "List all calendars",
						Action: cmd.CalendarsListAction,
					},
				},
			},
			{
				Name:  "events",
				Usage: "Manage events",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List events in a calendar",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "calendar-id",
								Usage:    "Calendar identifier",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "from",
								Usage:    "Start date (ISO 8601)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "to",
								Usage:    "End date (ISO 8601)",
								Required: true,
							},
						},
						Action: cmd.EventsListAction,
					},
					{
						Name:  "get",
						Usage: "Get a specific event",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "calendar-id",
								Usage:    "Calendar identifier",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "event-id",
								Usage:    "Event filename (basename including .ics)",
								Required: true,
							},
						},
						Action: cmd.EventsGetAction,
					},
					{
						Name:  "create",
						Usage: "Create a new event",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "calendar-id",
								Usage:    "Calendar identifier",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "title",
								Usage:    "Event title",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "start",
								Usage: "Start datetime (ISO 8601, e.g., 2025-01-15T10:00:00Z)",
							},
							&cli.StringFlag{
								Name:  "end",
								Usage: "End datetime (ISO 8601)",
							},
							&cli.StringFlag{
								Name:  "date",
								Usage: "Date for all-day event (YYYY-MM-DD)",
							},
							&cli.StringFlag{
								Name:  "description",
								Usage: "Event description",
							},
						},
						Action: cmd.EventsCreateAction,
					},
					{
						Name:  "update",
						Usage: "Update an existing event",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "calendar-id",
								Usage:    "Calendar identifier",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "event-id",
								Usage:    "Event filename (basename including .ics)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "title",
								Usage: "New title (optional, preserves existing if omitted)",
							},
							&cli.StringFlag{
								Name:  "start",
								Usage: "New start datetime (optional)",
							},
							&cli.StringFlag{
								Name:  "end",
								Usage: "New end datetime (optional)",
							},
							&cli.StringFlag{
								Name:  "date",
								Usage: "New date for all-day event (optional)",
							},
							&cli.StringFlag{
								Name:  "description",
								Usage: "New description (optional)",
							},
						},
						Action: cmd.EventsUpdateAction,
					},
					{
						Name:  "delete",
						Usage: "Delete an event",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "calendar-id",
								Usage:    "Calendar identifier",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "event-id",
								Usage:    "Event filename (basename including .ics)",
								Required: true,
							},
						},
						Action: cmd.EventsDeleteAction,
					},
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := cli.ShowRootCommandHelp(cmd.Root()); err != nil {
				return err
			}
			return internalerrors.NewFormattedError(errCommandRequired)
		},
	}
}

// Run executes the CLI application.
func Run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return run(ctx, os.Args, os.Stdout, os.Stderr)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	app := New()
	app.Writer = stdout
	app.ErrWriter = stderr

	if err := app.Run(ctx, args); err != nil {
		if !internalerrors.IsFormattedError(err) {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}

// GetFormatResolver returns a FormatResolver based on CLI context.
func GetFormatResolver(cmd *cli.Command) *out.FormatResolver {
	formatVal := cmd.String("format")
	return out.NewFormatResolver(formatVal)
}
