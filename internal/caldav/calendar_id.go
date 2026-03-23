package caldav

import (
	"path"
	"strconv"
	"strings"

	"github.com/emersion/go-webdav/caldav"
)

// CalendarIDFromHref derives a stable calendar_id from a calendar's href.
// The href is typically a server-relative path like "/dav.php/calendars/alice/personal/".
// The calendar_id is the last non-empty path component, with trailing slash stripped.
//
// Examples:
//   - "/dav.php/calendars/alice/personal/" -> "personal"
//   - "/dav.php/calendars/alice/work" -> "work"
//   - "/dav.php/calendars/alice/" -> error (empty basename)
//
// Collision handling: if two calendars would have the same calendar_id,
// the discovery process returns an error to the user rather than silently
// picking a winner, because stable identifiers are required for reliable automation.
func CalendarIDFromHref(href string) (string, error) {
	// Clean the href to remove trailing slashes
	href = strings.TrimSuffix(href, "/")

	// Extract the basename (last path component)
	basename := path.Base(href)
	if basename == "" || basename == "." || basename == "/" {
		return "", &CalendarIDError{Href: href, Reason: "empty calendar path component"}
	}

	return basename, nil
}

// ResolveCalendarID finds the calendar with the matching calendar_id
// from a list of discovered calendars.
func ResolveCalendarID(calendarID string, calendars []caldav.Calendar) (*caldav.Calendar, error) {
	for i := range calendars {
		cal := &calendars[i]
		id, err := CalendarIDFromHref(cal.Path)
		if err != nil {
			continue
		}
		if id == calendarID {
			return cal, nil
		}
	}
	return nil, &CalendarNotFoundError{CalendarID: calendarID}
}

// CheckDuplicateIDs checks if any calendars would have duplicate calendar_ids.
// Returns a map of calendar_id to count of occurrences.
func CheckDuplicateIDs(calendars []caldav.Calendar) map[string]int {
	idCounts := make(map[string]int)
	for _, cal := range calendars {
		id, err := CalendarIDFromHref(cal.Path)
		if err != nil {
			continue
		}
		idCounts[id]++
	}
	return idCounts
}

// CalendarIDError is returned when a calendar_id cannot be derived.
type CalendarIDError struct {
	Href   string
	Reason string
}

func (e *CalendarIDError) Error() string {
	if e.Href != "" {
		return "cannot derive calendar_id from href '" + e.Href + "': " + e.Reason
	}
	return "cannot derive calendar_id: " + e.Reason
}

// CalendarNotFoundError is returned when a requested calendar_id is not found.
type CalendarNotFoundError struct {
	CalendarID string
}

func (e *CalendarNotFoundError) Error() string {
	return "calendar with id '" + e.CalendarID + "' not found"
}

// DuplicateCalendarIDsError is returned when duplicate calendar_ids are detected.
type DuplicateCalendarIDsError struct {
	Duplicates map[string]int
}

func (e *DuplicateCalendarIDsError) Error() string {
	var sb strings.Builder
	sb.WriteString("duplicate calendar_ids detected: ")
	for id, count := range e.Duplicates {
		if count > 1 {
			sb.WriteString(id)
			sb.WriteString(" (")
			sb.WriteString(strconv.Itoa(count))
			sb.WriteString("), ")
		}
	}
	result := sb.String()
	return strings.TrimSuffix(result, ", ")
}
