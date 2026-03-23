package caldav

import (
	"context"
	"fmt"
	"strings"

	"github.com/emersion/go-webdav/caldav"
	"github.com/ksinistr/caldav-cli/internal/app"
)

// DiscoverCalendars performs the full discovery flow to find all calendars.
// It follows the CalDAV specification: principal -> calendar-home -> calendars.
func DiscoverCalendars(ctx context.Context, client *Client) (*app.CalendarsList, error) {
	// Step 1: Find the current user principal
	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover user principal: %w", err)
	}

	// Step 2: Find the calendar home set
	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	// Step 3: Find all calendars in the home set
	caldavCalendars, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	// Step 4: Check for duplicate calendar_ids
	duplicates := CheckDuplicateIDs(caldavCalendars)
	hasDupes := false
	for _, count := range duplicates {
		if count > 1 {
			hasDupes = true
			break
		}
	}
	if hasDupes {
		return nil, &DuplicateCalendarIDsError{Duplicates: duplicates}
	}

	// Step 5: Convert to app.Calendar DTOs with derived calendar_ids
	calendars := make([]app.Calendar, 0, len(caldavCalendars))
	for _, cal := range caldavCalendars {
		calendarID, err := CalendarIDFromHref(cal.Path)
		if err != nil {
			// Skip calendars we can't derive an ID for, but log the issue
			continue
		}

		// Use display name from calendar, or fall back to calendar_id
		name := cal.Name
		if name == "" {
			name = calendarID
		}

		calendars = append(calendars, app.Calendar{
			CalendarID: calendarID,
			Name:       name,
		})
	}

	return &app.CalendarsList{Calendars: calendars}, nil
}

// ResolveCalendarByPath finds a calendar by its server-relative path.
func ResolveCalendarByPath(path string, calendars []caldav.Calendar) (*caldav.Calendar, error) {
	for i := range calendars {
		if calendars[i].Path == path {
			return &calendars[i], nil
		}
	}
	return nil, fmt.Errorf("calendar not found at path '%s'", path)
}

// IsPrincipalURL checks if a URL looks like a principal URL.
func IsPrincipalURL(url string) bool {
	// Principal URLs typically don't end in specific calendar names
	// This is a heuristic check - in practice, the server tells us via properties
	return !strings.Contains(url, "/calendars/") ||
		strings.HasSuffix(url, "/principal/") ||
		strings.HasSuffix(url, "/user/")
}

// NormalizeHref ensures a server-relative href has consistent formatting.
func NormalizeHref(href string) string {
	href = strings.TrimSpace(href)
	if !strings.HasPrefix(href, "/") {
		href = "/" + href
	}
	// Remove duplicate slashes
	href = strings.ReplaceAll(href, "//", "/")
	return href
}
