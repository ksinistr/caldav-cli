package caldav

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/ksinistr/caldav-cli/internal/app"
)

// ErrUnsupportedRecurringEvent is returned when attempting to parse a recurring event.
// Recurring events (those with RRULE) are not supported in v1.
var ErrUnsupportedRecurringEvent = fmt.Errorf("recurring events are not supported in v1")

// ListEvents fetches events in a calendar within a time window.
// It uses calendar-query to filter events server-side, then parses each event.
func ListEvents(ctx context.Context, client *Client, calendarID string, from, to time.Time) (*app.EventsList, error) {
	// First, discover the calendar to get its path
	caldavClient := client.CalDAVClient()

	// We need to discover all calendars first to find the one matching calendarID
	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	// Resolve the calendar by ID
	calendar, err := ResolveCalendarID(calendarID, cals)
	if err != nil {
		return nil, fmt.Errorf("calendar '%s' not found: %w", calendarID, err)
	}

	// Build time range filter for CalDAV query
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name:  "VCALENDAR",
			Props: []string{"GETETAG"},
			Comps: []caldav.CalendarCompRequest{
				{
					Name:  "VEVENT",
					Props: []string{"UID", "SUMMARY", "DESCRIPTION", "DTSTART", "DTEND", "DURATION"},
				},
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{
				{
					Name:  "VEVENT",
					Start: from,
					End:   to,
				},
			},
		},
	}

	// Query the calendar for events in the time range
	eventObjs, err := caldavClient.QueryCalendar(ctx, calendar.Path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar: %w", err)
	}

	// Parse each event from the iCalendar data
	events := make([]app.Event, 0, len(eventObjs))
	for _, eventObj := range eventObjs {
		// Parse the iCalendar data directly from the calendar object
		event, err := parseEventFromCalendar(eventObj.Data, calendarID, eventObj.Path)
		if err != nil {
			// Skip unsupported or malformed events but continue processing
			// Recurring events (ErrUnsupportedRecurringEvent) are skipped in v1
			continue
		}
		if event != nil {
			events = append(events, *event)
		}
	}

	return &app.EventsList{
		CalendarID: calendarID,
		From:       from.Format(time.RFC3339),
		To:         to.Format(time.RFC3339),
		Events:     events,
	}, nil
}

// parseEventFromCalendar parses a single VEVENT from an iCalendar into an app.Event.
// It returns nil for unsupported event types (like recurring events).
func parseEventFromCalendar(cal *ical.Calendar, calendarID, href string) (*app.Event, error) {
	for _, component := range cal.Children {
		if component.Name == ical.CompEvent {
			return parseVEVENT(component, calendarID, href)
		}
	}

	return nil, fmt.Errorf("no VEVENT found in iCalendar")
}

// parseVEVENT parses a single VEVENT component.
func parseVEVENT(comp *ical.Component, calendarID, href string) (*app.Event, error) {
	event := &app.Event{
		CalendarID: calendarID,
		EventID:    eventIDFromHref(href),
		AllDay:     false,
	}

	// Extract UID
	if uid := comp.Props.Get(ical.PropUID); uid != nil {
		event.UID = uid.Value
	}

	// Extract Summary (title)
	if summary := comp.Props.Get(ical.PropSummary); summary != nil {
		event.Title = summary.Value
	}

	// Extract Description
	if desc := comp.Props.Get(ical.PropDescription); desc != nil {
		event.Description = desc.Value
	}

	// Check for recurring events - we skip them in v1
	// Check for RRULE
	if comp.Props.Get(ical.PropRecurrenceRule) != nil {
		return nil, ErrUnsupportedRecurringEvent
	}

	// Parse DTSTART
	dtStart := comp.Props.Get(ical.PropDateTimeStart)
	if dtStart == nil {
		return nil, fmt.Errorf("event missing DTSTART")
	}

	// Parse DTEND or DURATION
	dtEnd := comp.Props.Get(ical.PropDateTimeEnd)
	duration := comp.Props.Get(ical.PropDuration)

	// Check if this is an all-day event
	// All-day events have DATE values (not DATE-TIME)
	// In go-ical, date-only values are indicated by the VALUE=DATE parameter
	isAllDay := dtStart.Params != nil && dtStart.Params.Get("VALUE") == "DATE"
	// Also check by seeing if the value can be parsed as a date (8 digits)
	if !isAllDay && len(dtStart.Value) == 8 {
		if _, err := time.Parse("20060102", dtStart.Value); err == nil {
			isAllDay = true
		}
	}

	if isAllDay {
		event.AllDay = true
		// Parse all-day start
		startDate, err := time.Parse("20060102", dtStart.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse all-day start: %w", err)
		}
		event.Start = startDate

		// For all-day events, DTEND is exclusive (the day after the event ends)
		// Or use DURATION to calculate the end
		if dtEnd != nil {
			endDate, err := time.Parse("20060102", dtEnd.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse all-day end: %w", err)
			}
			// DTEND for all-day is exclusive, so subtract one day to get the actual end date
			event.End = endDate.Add(-24 * time.Hour)
		} else if duration != nil {
			dur, err := parseDuration(duration.Value)
			if err == nil {
				event.End = startDate.Add(dur)
			} else {
				// Default to same day for single-day all-day events
				event.End = startDate
			}
		} else {
			// Single day all-day event
			event.End = startDate
		}
	} else {
		// Timed event
		// DTSTART might be in UTC or with a TZID
		startTime, err := parseDateTime(dtStart)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start time: %w", err)
		}
		event.Start = startTime

		if dtEnd != nil {
			endTime, err := parseDateTime(dtEnd)
			if err != nil {
				return nil, fmt.Errorf("failed to parse end time: %w", err)
			}
			event.End = endTime
		} else if duration != nil {
			dur, err := parseDuration(duration.Value)
			if err == nil {
				event.End = startTime.Add(dur)
			} else {
				event.End = startTime
			}
		} else {
			return nil, fmt.Errorf("event missing DTEND or DURATION")
		}
	}

	// Validate that start <= end
	if event.End.Before(event.Start) {
		return nil, fmt.Errorf("event end time (%s) is before start time (%s)", event.End, event.Start)
	}

	return event, nil
}

// eventIDFromHref extracts the event_id (basename with .ics) from a full href.
// Uses path.Base to sanitize and extract the last component, protecting against
// path traversal sequences like "../".
func eventIDFromHref(href string) string {
	// path.Base cleans the path and returns the last component
	// This handles "../" and other traversal sequences safely
	basename := path.Base(href)
	if basename == "" || basename == "." || basename == "/" {
		// Fallback to original if path.Base returns empty
		return href
	}
	return basename
}

// validateEventID validates that an event_id is a safe basename.
// Returns an error if the event_id contains path traversal sequences or separators.
func validateEventID(eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event_id cannot be empty")
	}
	// Explicitly reject dot-segments that path.Base returns unchanged
	// but which are still dangerous for path traversal
	if eventID == "." || eventID == ".." {
		return fmt.Errorf("event_id cannot be '.' or '..': %s", eventID)
	}
	// Check for path separators that could allow traversal
	if strings.Contains(eventID, "/") || strings.Contains(eventID, "\\") {
		return fmt.Errorf("event_id cannot contain path separators: %s", eventID)
	}
	// Verify that path.Base returns the same value (detects traversal attempts)
	// This catches cases like "../etc" where path.Base returns "etc"
	cleaned := path.Base(eventID)
	if cleaned != eventID {
		return fmt.Errorf("event_id must be a basename, got: %s (normalized to: %s)", eventID, cleaned)
	}
	return nil
}

// parseDateTime parses an iCalendar DATE-TIME value.
// It handles UTC times (with Z suffix), local times (floating time),
// and times with an explicit TZID parameter.
func parseDateTime(prop *ical.Prop) (time.Time, error) {
	val := prop.Value

	// Check for TZID parameter first
	if prop.Params != nil {
		if tzid := prop.Params.Get("TZID"); tzid != "" {
			return parseDateTimeInTZID(val, tzid)
		}
	}

	// iCalendar basic format: 20260325T090000 or 20260325T090000Z
	format := "20060102T150405"
	if strings.HasSuffix(val, "Z") {
		t, err := time.Parse(format+"Z", val)
		if err == nil {
			return t, nil
		}
	} else {
		t, err := time.Parse(format, val)
		if err == nil {
			return t, nil
		}
	}

	// Try RFC3339 format as fallback
	if strings.HasSuffix(val, "Z") {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t, nil
		}
	} else {
		t, err := time.Parse(time.RFC3339, val+"Z")
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse datetime: %s", val)
}

// parseDateTimeInTZID parses a datetime value with an explicit timezone ID.
// The iCalendar format with TZID does not include a "Z" suffix - the timezone
// is specified separately via the TZID parameter.
func parseDateTimeInTZID(val, tzid string) (time.Time, error) {
	// Load the timezone location
	loc, err := time.LoadLocation(tzid)
	if err != nil {
		// Non-IANA timezone IDs (VTIMEZONE-defined IDs, Windows timezone names, etc.)
		// cannot be handled by time.LoadLocation. Full VTIMEZONE parsing is out of
		// scope for v1, so we reject these events to avoid silently corrupting the data.
		// Accepting and re-serializing would misinterpret the local time as UTC, shifting
		// the event time by several hours on update.
		return time.Time{}, fmt.Errorf("non-IANA timezone ID '%s' is not supported in v1 (requires VTIMEZONE parsing)", tzid)
	}

	// Parse the time value in the specified timezone
	format := "20060102T150405"
	t, err := time.ParseInLocation(format, val, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse datetime with TZID %s: %w", tzid, err)
	}

	return t, nil
}

// parseDuration parses an iCalendar DURATION value.
// Format: P[n]DT[n]H[n]M[n]S
func parseDuration(dur string) (time.Duration, error) {
	// Simple parser for common duration formats
	// e.g., PT30M for 30 minutes, P1D for 1 day, PT1H for 1 hour
	var days, hours, minutes, seconds int

	rest := strings.TrimPrefix(dur, "P")

	// Parse days
	if idx := strings.Index(rest, "D"); idx > 0 {
		if n, err := fmt.Sscanf(rest[:idx], "%d", &days); err != nil || n != 1 {
			return 0, fmt.Errorf("invalid duration format: cannot parse days: %w", err)
		}
		rest = rest[idx+1:]
	}

	// Parse time component
	rest = strings.TrimPrefix(rest, "T")

	// Parse hours
	if idx := strings.Index(rest, "H"); idx > 0 {
		_, _ = fmt.Sscanf(rest[:idx], "%d", &hours)
		rest = rest[idx+1:]
	}

	// Parse minutes
	if idx := strings.Index(rest, "M"); idx > 0 {
		_, _ = fmt.Sscanf(rest[:idx], "%d", &minutes)
		rest = rest[idx+1:]
	}

	// Parse seconds
	if idx := strings.Index(rest, "S"); idx > 0 {
		_, _ = fmt.Sscanf(rest[:idx], "%d", &seconds)
	}

	totalSeconds := days*86400 + hours*3600 + minutes*60 + seconds
	return time.Duration(totalSeconds) * time.Second, nil
}

// EventIDFromHref derives an event_id from a full event href.
// This is exported for use in other packages.
func EventIDFromHref(href string) string {
	return eventIDFromHref(href)
}

// CreateEventInput contains the parameters for creating a new event.
type CreateEventInput struct {
	CalendarID  string
	Title       string
	Description string
	// For timed events
	Start *time.Time
	End   *time.Time
	// For all-day events
	Date *time.Time
}

// CreateEvent creates a new event on the calendar.
// It generates a UID and event filename, creates the iCalendar data,
// and puts it on the server.
func CreateEvent(ctx context.Context, client *Client, input CreateEventInput) (*app.MutationResult, error) {
	// Validate input
	if input.CalendarID == "" {
		return nil, fmt.Errorf("calendar_id is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Determine if this is a timed or all-day event
	isAllDay := input.Date != nil
	isTimed := input.Start != nil && input.End != nil

	if !isAllDay && !isTimed {
		return nil, fmt.Errorf("either --date (for all-day) or --start and --end (for timed) must be provided")
	}

	if isAllDay && isTimed {
		return nil, fmt.Errorf("cannot use --date with --start/--end flags together")
	}

	// Discover the calendar
	caldavClient := client.CalDAVClient()

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	calendar, err := ResolveCalendarID(input.CalendarID, cals)
	if err != nil {
		return nil, fmt.Errorf("calendar '%s' not found: %w", input.CalendarID, err)
	}

	// Generate UID
	uid := generateUID()

	// Generate event filename (event_id)
	eventID := generateEventID(input.Title)

	// Construct the event href
	eventHref := calendar.Path
	if !strings.HasSuffix(eventHref, "/") {
		eventHref += "/"
	}
	eventHref += eventID

	// Create the iCalendar data
	cal, err := buildICalendar(uid, input.Title, input.Description, isAllDay, input.Start, input.End, input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to build iCalendar: %w", err)
	}

	// Put the event on the server
	_, err = caldavClient.PutCalendarObject(ctx, eventHref, cal)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return &app.MutationResult{
		Action:     "created",
		CalendarID: input.CalendarID,
		EventID:    eventID,
		UID:        uid,
		Title:      input.Title,
		Message:    fmt.Sprintf("Event '%s' created successfully", input.Title),
	}, nil
}

// generateUID generates a unique ID for a new event.
func generateUID() string {
	// Format: YYYYMMDDTHHMMSSZ-<random>@<host>
	// Use timestamp and a random component for uniqueness
	now := time.Now().UTC()
	timestamp := now.Format("20060102T150405Z")
	randomStr := fmt.Sprintf("%08x", now.UnixNano()%0x100000000)
	return timestamp + "-" + randomStr + "@caldav-cli"
}

// generateEventID generates an event filename from a title.
func generateEventID(title string) string {
	// Format: YYYY-MM-DD-HHMMSS-<nanoseconds>-<sanitized-title>.ics
	// Include nanosecond component to ensure uniqueness for same-title events created in the same second
	now := time.Now().UTC()
	datePart := now.Format("2006-01-02")
	timePart := now.Format("150405")
	nanoPart := fmt.Sprintf("%09d", now.Nanosecond())

	// Sanitize the title: lowercase, replace spaces with hyphens, remove special chars
	sanitized := strings.ToLower(title)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")

	// Remove any character that isn't alphanumeric or hyphen
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	sanitized = result.String()
	// Limit length and truncate
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	// Ensure we have something
	if sanitized == "" {
		sanitized = "event"
	}

	return fmt.Sprintf("%s-%s-%s-%s.ics", datePart, timePart, nanoPart, sanitized)
}

// buildICalendar creates an iCalendar object for a single event.
func buildICalendar(uid, title, description string, allDay bool, start, end, date *time.Time) (*ical.Calendar, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//caldav-cli//caldav-cli//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetText(ical.PropSummary, title)

	if description != "" {
		event.Props.SetText(ical.PropDescription, description)
	}

	// Set DTSTAMP to now
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())

	if allDay && date != nil {
		// All-day event
		startDate := date.Format("20060102")
		endDate := date.Add(24 * time.Hour).Format("20060102") // DTEND is exclusive

		// Create DTSTART with VALUE=DATE parameter
		dtStart := ical.NewProp(ical.PropDateTimeStart)
		dtStart.Value = startDate
		dtStart.Params = make(ical.Params)
		dtStart.Params.Set("VALUE", "DATE")
		event.Props.Set(dtStart)

		// Create DTEND with VALUE=DATE parameter
		dtEnd := ical.NewProp(ical.PropDateTimeEnd)
		dtEnd.Value = endDate
		dtEnd.Params = make(ical.Params)
		dtEnd.Params.Set("VALUE", "DATE")
		event.Props.Set(dtEnd)
	} else if start != nil && end != nil {
		// Timed event
		event.Props.SetDateTime(ical.PropDateTimeStart, *start)
		event.Props.SetDateTime(ical.PropDateTimeEnd, *end)
	}

	cal.Children = append(cal.Children, event.Component)

	return cal, nil
}

// GetEvent fetches a single event by its calendar_id and event_id.
// It discovers the calendar to get its path, constructs the event href,
// and fetches the event object from the server.
func GetEvent(ctx context.Context, client *Client, calendarID, eventID string) (*app.EventDetail, error) {
	// First, discover the calendar to get its path
	caldavClient := client.CalDAVClient()

	// We need to discover all calendars first to find the one matching calendarID
	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	// Resolve the calendar by ID
	calendar, err := ResolveCalendarID(calendarID, cals)
	if err != nil {
		return nil, fmt.Errorf("calendar '%s' not found: %w", calendarID, err)
	}

	// Validate eventID is a safe basename to prevent path traversal
	if err := validateEventID(eventID); err != nil {
		return nil, fmt.Errorf("invalid event_id: %w", err)
	}

	// Construct the event href from calendar path and event_id
	// The event_id is the basename including .ics, so we join it with the calendar path
	eventHref := calendar.Path
	if !strings.HasSuffix(eventHref, "/") {
		eventHref += "/"
	}
	eventHref += eventID

	// Fetch the event object from the server
	eventObj, err := caldavClient.GetCalendarObject(ctx, eventHref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	// Parse the iCalendar data into an app.Event
	// The CalendarObject.Data is already an *ical.Calendar
	event, err := parseEventFromCalendar(eventObj.Data, calendarID, eventHref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	return &app.EventDetail{
		Event: *event,
	}, nil
}

// UpdateEventInput contains the parameters for updating an existing event.
// Omitted fields preserve their existing values.
type UpdateEventInput struct {
	CalendarID string
	EventID    string
	// Optional fields to update
	Title       *string
	Description *string
	// For timed events - omit to preserve existing
	Start *time.Time
	End   *time.Time
	// For all-day events - omit to preserve existing
	Date *time.Time
}

// UpdateEvent updates an existing event on the calendar.
// It loads the current event, applies the updates, and writes it back.
func UpdateEvent(ctx context.Context, client *Client, input UpdateEventInput) (*app.MutationResult, error) {
	// Validate input
	if input.CalendarID == "" {
		return nil, fmt.Errorf("calendar_id is required")
	}
	if input.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	// Validate eventID is a safe basename to prevent path traversal
	if err := validateEventID(input.EventID); err != nil {
		return nil, fmt.Errorf("invalid event_id: %w", err)
	}

	// Discover the calendar to get its path
	caldavClient := client.CalDAVClient()

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	calendar, err := ResolveCalendarID(input.CalendarID, cals)
	if err != nil {
		return nil, fmt.Errorf("calendar '%s' not found: %w", input.CalendarID, err)
	}

	// Construct the event href
	eventHref := calendar.Path
	if !strings.HasSuffix(eventHref, "/") {
		eventHref += "/"
	}
	eventHref += input.EventID

	// Fetch the current event
	eventObj, err := caldavClient.GetCalendarObject(ctx, eventHref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	// Parse the current event to get its properties
	currentEvent, err := parseEventFromCalendar(eventObj.Data, input.CalendarID, eventHref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current event: %w", err)
	}

	// Find the VEVENT component in the current calendar
	var vevent *ical.Component
	for _, comp := range eventObj.Data.Children {
		if comp.Name == ical.CompEvent {
			vevent = comp
			break
		}
	}

	if vevent == nil {
		return nil, fmt.Errorf("no VEVENT found in current event")
	}

	// Extract the UID (must be preserved)
	uid := vevent.Props.Get(ical.PropUID)
	if uid == nil {
		return nil, fmt.Errorf("event missing UID")
	}

	// Extract the existing SEQUENCE for incrementing (RFC 5545 requirement)
	var existingSeq int
	seqProp := vevent.Props.Get(ical.PropSequence)
	if seqProp != nil && seqProp.Value != "" {
		existingSeq, _ = strconv.Atoi(seqProp.Value)
	}

	// Determine the new values, preserving existing if not provided
	newTitle := currentEvent.Title
	if input.Title != nil {
		newTitle = *input.Title
	}

	newDescription := currentEvent.Description
	if input.Description != nil {
		newDescription = *input.Description
	}

	// Determine if this is an all-day event and the new times
	isAllDay := currentEvent.AllDay
	var newStart, newEnd *time.Time

	// If date is provided, it's an all-day update
	if input.Date != nil {
		if input.Start != nil || input.End != nil {
			return nil, fmt.Errorf("cannot use --date with --start/--end flags together")
		}
		isAllDay = true
		newStart = input.Date
		// For all-day events, end is the same as start (single day)
		newEnd = input.Date
	} else if input.Start != nil || input.End != nil {
		// Timed event update
		if input.Start != nil && input.End == nil {
			return nil, fmt.Errorf("--end required when --start is provided")
		}
		if input.Start == nil && input.End != nil {
			return nil, fmt.Errorf("--start required when --end is provided")
		}
		isAllDay = false
		newStart = input.Start
		newEnd = input.End
	} else {
		// Preserve existing timing
		newStart = &currentEvent.Start
		newEnd = &currentEvent.End
	}

	// Build the updated iCalendar
	cal, err := buildUpdatedICalendar(uid.Value, newTitle, newDescription, isAllDay, newStart, newEnd, existingSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to build updated iCalendar: %w", err)
	}

	// Write the updated event back to the server
	_, err = caldavClient.PutCalendarObject(ctx, eventHref, cal)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return &app.MutationResult{
		Action:     "updated",
		CalendarID: input.CalendarID,
		EventID:    input.EventID,
		UID:        uid.Value,
		Title:      newTitle,
		Message:    fmt.Sprintf("Event '%s' updated successfully", newTitle),
	}, nil
}

// buildUpdatedICalendar creates an updated iCalendar object for an existing event.
// It preserves the UID and updates the specified fields.
// The sequence parameter is the existing SEQUENCE value which will be incremented per RFC 5545.
func buildUpdatedICalendar(uid, title, description string, allDay bool, start, end *time.Time, sequence int) (*ical.Calendar, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//caldav-cli//caldav-cli//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetText(ical.PropSummary, title)

	if description != "" {
		event.Props.SetText(ical.PropDescription, description)
	}

	// Set DTSTAMP to now
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())

	// Increment SEQUENCE per RFC 5545 for proper update tracking
	event.Props.SetText(ical.PropSequence, strconv.Itoa(sequence+1))

	if allDay && start != nil && end != nil {
		// All-day event
		startDate := start.Format("20060102")
		endDate := end.Add(24 * time.Hour).Format("20060102") // DTEND is exclusive

		// Create DTSTART with VALUE=DATE parameter
		dtStart := ical.NewProp(ical.PropDateTimeStart)
		dtStart.Value = startDate
		dtStart.Params = make(ical.Params)
		dtStart.Params.Set("VALUE", "DATE")
		event.Props.Set(dtStart)

		// Create DTEND with VALUE=DATE parameter
		dtEnd := ical.NewProp(ical.PropDateTimeEnd)
		dtEnd.Value = endDate
		dtEnd.Params = make(ical.Params)
		dtEnd.Params.Set("VALUE", "DATE")
		event.Props.Set(dtEnd)
	} else if start != nil && end != nil {
		// Timed event
		event.Props.SetDateTime(ical.PropDateTimeStart, *start)
		event.Props.SetDateTime(ical.PropDateTimeEnd, *end)
	}

	cal.Children = append(cal.Children, event.Component)

	return cal, nil
}

// DeleteEvent deletes an event from the calendar.
func DeleteEvent(ctx context.Context, client *Client, calendarID, eventID string) (*app.MutationResult, error) {
	// Validate input
	if calendarID == "" {
		return nil, fmt.Errorf("calendar_id is required")
	}
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	// Validate eventID is a safe basename to prevent path traversal
	if err := validateEventID(eventID); err != nil {
		return nil, fmt.Errorf("invalid event_id: %w", err)
	}

	// Discover the calendar to get its path
	webdavClient := client.WebDAVClient()

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	calendar, err := ResolveCalendarID(calendarID, cals)
	if err != nil {
		return nil, fmt.Errorf("calendar '%s' not found: %w", calendarID, err)
	}

	// Construct the event href
	eventHref := calendar.Path
	if !strings.HasSuffix(eventHref, "/") {
		eventHref += "/"
	}
	eventHref += eventID

	// Delete the event using WebDAV RemoveAll
	err = webdavClient.RemoveAll(ctx, eventHref)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &app.MutationResult{
		Action:     "deleted",
		CalendarID: calendarID,
		EventID:    eventID,
		Message:    fmt.Sprintf("Event '%s' deleted successfully", eventID),
	}, nil
}
