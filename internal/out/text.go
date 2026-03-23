package out

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ksinistr/caldav-cli/internal/app"
)

// textRenderer outputs data in human-readable text format.
type textRenderer struct{}

// Render outputs data in human-readable text format.
func (r *textRenderer) Render(ctx context.Context, w io.Writer, data any) error {
	switch v := data.(type) {
	case *app.CalendarsList:
		return renderCalendarsList(w, v)
	case *app.EventsList:
		return renderEventsList(w, v)
	case *app.EventDetail:
		return renderEventDetail(w, v)
	case *app.MutationResult:
		return renderMutationResult(w, v)
	case ErrorEnvelope:
		return renderTextError(w, v)
	case *app.Error:
		return renderTextAppError(w, v)
	default:
		// Fallback for unknown types
		_, err := fmt.Fprintf(w, "%v\n", data)
		return err
	}
}

// RenderText renders data in human-readable text format.
func RenderText(ctx context.Context, w io.Writer, data any) error {
	return (&textRenderer{}).Render(ctx, w, data)
}

func renderCalendarsList(w io.Writer, cl *app.CalendarsList) error {
	if len(cl.Calendars) == 0 {
		_, err := fmt.Fprintln(w, "No calendars found.")
		return err
	}
	_, err := fmt.Fprintf(w, "Found %d calendar(s):\n", len(cl.Calendars))
	if err != nil {
		return err
	}
	for _, cal := range cl.Calendars {
		_, err := fmt.Fprintf(w, "  - %s (id: %s)\n", cal.Name, cal.CalendarID)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderEventsList(w io.Writer, el *app.EventsList) error {
	if len(el.Events) == 0 {
		_, err := fmt.Fprintf(w, "No events found in calendar '%s' for the specified time range.\n", el.CalendarID)
		return err
	}
	_, err := fmt.Fprintf(w, "Events in '%s' (%s to %s):\n", el.CalendarID, el.From, el.To)
	if err != nil {
		return err
	}
	for _, e := range el.Events {
		if e.AllDay {
			_, err = fmt.Fprintf(w, "  - %s: %s (all day, id: %s)\n",
				formatDate(e.Start), e.Title, e.EventID)
		} else {
			_, err = fmt.Fprintf(w, "  - %s: %s (id: %s)\n",
				formatDateTime(e.Start), e.Title, e.EventID)
		}
		if err != nil {
			return err
		}
		if e.Description != "" {
			_, err = fmt.Fprintf(w, "    Description: %s\n", e.Description)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func renderEventDetail(w io.Writer, ed *app.EventDetail) error {
	e := ed.Event
	_, err := fmt.Fprintf(w, "Event: %s\n", e.Title)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  Calendar ID: %s\n", e.CalendarID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  Event ID: %s\n", e.EventID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  UID: %s\n", e.UID)
	if err != nil {
		return err
	}
	if e.AllDay {
		_, err = fmt.Fprintf(w, "  Date: %s (all day)\n", formatDate(e.Start))
	} else {
		_, err = fmt.Fprintf(w, "  Start: %s\n", formatDateTime(e.Start))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "  End: %s\n", formatDateTime(e.End))
	}
	if err != nil {
		return err
	}
	if e.Description != "" {
		_, err = fmt.Fprintf(w, "  Description: %s\n", e.Description)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderMutationResult(w io.Writer, mr *app.MutationResult) error {
	_, err := fmt.Fprintf(w, "%s: %s\n", mr.Action, mr.Message)
	if err != nil {
		return err
	}
	if mr.CalendarID != "" {
		_, err = fmt.Fprintf(w, "  Calendar ID: %s\n", mr.CalendarID)
		if err != nil {
			return err
		}
	}
	if mr.EventID != "" {
		_, err = fmt.Fprintf(w, "  Event ID: %s\n", mr.EventID)
		if err != nil {
			return err
		}
	}
	if mr.UID != "" {
		_, err = fmt.Fprintf(w, "  UID: %s\n", mr.UID)
		if err != nil {
			return err
		}
	}
	if mr.Title != "" {
		_, err = fmt.Fprintf(w, "  Title: %s\n", mr.Title)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderTextError(w io.Writer, ee ErrorEnvelope) error {
	_, err := fmt.Fprintf(w, "Error: %s\n", ee.Error)
	return err
}

func renderTextAppError(w io.Writer, ae *app.Error) error {
	if ae.Code != "" {
		_, err := fmt.Fprintf(w, "Error [%s]: %s\n", ae.Code, ae.Message)
		return err
	}
	_, err := fmt.Fprintf(w, "Error: %s\n", ae.Message)
	return err
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func formatDateTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
