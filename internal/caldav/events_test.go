package caldav

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/ksinistr/caldav-cli/internal/config"
)

type calendarQueryXML struct {
	XMLName xml.Name       `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Filter  calendarFilter `xml:"filter"`
}

type calendarFilter struct {
	CompFilter calendarCompFilter `xml:"comp-filter"`
}

type calendarCompFilter struct {
	Name        string               `xml:"name,attr"`
	TimeRange   *struct{}            `xml:"time-range"`
	CompFilters []calendarCompFilter `xml:"comp-filter"`
}

// mockEventServer creates a test server that responds with event data.
func mockEventServer(t *testing.T, events map[string]string) *httptest.Server {
	return mockEventServerWithReportValidator(t, events, nil)
}

func mockEventServerWithReportValidator(t *testing.T, events map[string]string, validateReport func(*testing.T, []byte)) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && (r.URL.Path == "/dav.php" || r.URL.Path == "/dav.php/"):
			// Current user principal discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<D:current-user-principal>
<D:href>/dav.php/principal/</D:href>
</D:current-user-principal>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/principal/":
			// Calendar home set discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<C:calendar-home-set>
<D:href>/dav.php/calendars/alice/</D:href>
</C:calendar-home-set>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/calendars/alice/":
			// Calendar collection discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:A="http://apple.com/ns/ical/">
<D:response>
<D:href>/dav.php/calendars/alice/personal/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Personal Calendar</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "REPORT" && r.URL.Path == "/dav.php/calendars/alice/personal/":
			if validateReport != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read report body: %v", err)
				}
				validateReport(t, body)
			}

			// Calendar query - return event data directly
			var eventPaths strings.Builder
			eventPaths.WriteString(`<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">`)

			// Add mock events with full calendar data
			for eventPath, eventData := range events {
				// Escape the iCalendar data for XML
				xmlData := strings.ReplaceAll(eventData, "&", "&amp;")
				xmlData = strings.ReplaceAll(xmlData, "<", "&lt;")
				xmlData = strings.ReplaceAll(xmlData, ">", "&gt;")
				xmlData = strings.ReplaceAll(xmlData, "\"", "&quot;")

				eventPaths.WriteString(`<D:response>
<D:href>/dav.php/calendars/alice/personal/`)
				eventPaths.WriteString(eventPath)
				eventPaths.WriteString(`</D:href>
<D:propstat>
<D:prop>
<D:getetag>"test-etag"</D:getetag>
<C:calendar-data>`)
				eventPaths.WriteString(xmlData)
				eventPaths.WriteString(`</C:calendar-data>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>`)
			}

			eventPaths.WriteString(`</D:multistatus>`)
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, eventPaths.String())

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/dav.php/calendars/alice/personal/"):
			// Return the event content
			eventPath := strings.TrimPrefix(r.URL.Path, "/dav.php/calendars/alice/personal/")
			if eventData, ok := events[eventPath]; ok {
				w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, eventData)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestListEventsBuildsVEVENTTimeRangeFilter(t *testing.T) {
	events := map[string]string{
		"team-sync.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServerWithReportValidator(t, events, func(t *testing.T, body []byte) {
		t.Helper()

		var query calendarQueryXML
		if err := xml.Unmarshal(body, &query); err != nil {
			t.Fatalf("failed to parse calendar-query: %v\nbody=%s", err, string(body))
		}

		if query.Filter.CompFilter.Name != "VCALENDAR" {
			t.Fatalf("top-level comp-filter = %q, want VCALENDAR", query.Filter.CompFilter.Name)
		}
		if query.Filter.CompFilter.TimeRange != nil {
			t.Fatal("top-level VCALENDAR comp-filter must not include time-range")
		}
		if len(query.Filter.CompFilter.CompFilters) != 1 {
			t.Fatalf("got %d nested comp-filters, want 1", len(query.Filter.CompFilter.CompFilters))
		}

		eventFilter := query.Filter.CompFilter.CompFilters[0]
		if eventFilter.Name != "VEVENT" {
			t.Fatalf("nested comp-filter = %q, want VEVENT", eventFilter.Name)
		}
		if eventFilter.TimeRange == nil {
			t.Fatal("VEVENT comp-filter must include time-range")
		}
	})
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	if _, err := ListEvents(ctx, client, "personal", from, to); err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
}

// Test timed event listing
func TestListEventsTimed(t *testing.T) {
	events := map[string]string{
		"team-sync.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	result, err := ListEvents(ctx, client, "personal", from, to)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if result.CalendarID != "personal" {
		t.Errorf("CalendarID = %v, want 'personal'", result.CalendarID)
	}

	if len(result.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(result.Events))
	}

	event := result.Events[0]
	if event.EventID != "team-sync.ics" {
		t.Errorf("EventID = %v, want 'team-sync.ics'", event.EventID)
	}
	if event.UID != "20260325T090000Z-team-sync@example.com" {
		t.Errorf("UID = %v, want '20260325T090000Z-team-sync@example.com'", event.UID)
	}
	if event.Title != "Team Sync" {
		t.Errorf("Title = %v, want 'Team Sync'", event.Title)
	}
	if event.Description != "Weekly team sync meeting" {
		t.Errorf("Description = %v, want 'Weekly team sync meeting'", event.Description)
	}
	if event.AllDay {
		t.Errorf("AllDay = %v, want false", event.AllDay)
	}
}

// Test all-day event listing
func TestListEventsAllDay(t *testing.T) {
	events := map[string]string{
		"day-off.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260327-dayoff@example.com
DTSTART;VALUE=DATE:20260327
DTEND;VALUE=DATE:20260328
SUMMARY:Day Off
DESCRIPTION:Vacation day
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	result, err := ListEvents(ctx, client, "personal", from, to)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(result.Events))
	}

	event := result.Events[0]
	if event.EventID != "day-off.ics" {
		t.Errorf("EventID = %v, want 'day-off.ics'", event.EventID)
	}
	if event.Title != "Day Off" {
		t.Errorf("Title = %v, want 'Day Off'", event.Title)
	}
	if !event.AllDay {
		t.Errorf("AllDay = %v, want true", event.AllDay)
	}
	// Check that start and end are on the same day for single-day all-day event
	if event.Start.Year() != 2026 || event.Start.Month() != 3 || event.Start.Day() != 27 {
		t.Errorf("Start = %v, want 2026-03-27", event.Start)
	}
}

// Test empty result when no events in range
func TestListEventsEmpty(t *testing.T) {
	// Server with no events
	events := map[string]string{}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	result, err := ListEvents(ctx, client, "personal", from, to)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if len(result.Events) != 0 {
		t.Errorf("got %d events, want 0", len(result.Events))
	}
}

// Test recurring events are skipped
func TestListEventsSkipsRecurring(t *testing.T) {
	events := map[string]string{
		"daily-standup.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:daily-standup@example.com
DTSTART:20260325T090000Z
DTEND:20260325T091500Z
SUMMARY:Daily Standup
RRULE:FREQ=DAILY
END:VEVENT
END:VCALENDAR`,
		// Also add a non-recurring event to verify we still get some results
		"onetime.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:onetime@example.com
DTSTART:20260325T100000Z
DTEND:20260325T103000Z
SUMMARY:One-time Meeting
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	result, err := ListEvents(ctx, client, "personal", from, to)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	// Should only get the non-recurring event
	if len(result.Events) != 1 {
		t.Fatalf("got %d events, want 1 (recurring should be skipped)", len(result.Events))
	}

	if result.Events[0].Title != "One-time Meeting" {
		t.Errorf("Title = %v, want 'One-time Meeting' (recurring should be skipped)", result.Events[0].Title)
	}
}

// Test invalid calendar ID
func TestListEventsInvalidCalendarID(t *testing.T) {
	events := map[string]string{}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	from, _ := time.Parse(time.RFC3339, "2026-03-24T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-31T23:59:59Z")

	_, err = ListEvents(ctx, client, "nonexistent", from, to)
	if err == nil {
		t.Error("expected error for invalid calendar ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}
}

// Test parseDuration
func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		dur     string
		wantSec int64
	}{
		{"30 minutes", "PT30M", 1800},
		{"1 hour", "PT1H", 3600},
		{"1 day", "P1D", 86400},
		{"1 hour 30 minutes", "PT1H30M", 5400},
		{"90 minutes", "PT90M", 5400},
		{"1 day 2 hours", "P1DT2H", 93600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.dur)
			if err != nil {
				t.Fatalf("parseDuration() error = %v", err)
			}
			if got.Seconds() != float64(tt.wantSec) {
				t.Errorf("parseDuration() = %v seconds, want %v", got.Seconds(), tt.wantSec)
			}
		})
	}
}

// Test eventIDFromHref
func TestEventIDFromHref(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{"/dav.php/calendars/alice/personal/team-sync.ics", "team-sync.ics"},
		{"/dav.php/calendars/alice/personal/event.ics", "event.ics"},
		{"event.ics", "event.ics"},
		{"https://example.com/calendars/personal/meeting.ics", "meeting.ics"},
	}

	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			if got := eventIDFromHref(tt.href); got != tt.want {
				t.Errorf("eventIDFromHref() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test parseDateTime
func TestParseDateTime(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		tzid    string
		wantErr bool
	}{
		{"UTC time with Z", "20260325T090000Z", "", false},
		{"UTC time without Z", "20260325T090000", "", false},
		{"RFC3339 format", "2026-03-25T09:00:00Z", "", false},
		{"IANA timezone America/New_York", "20260325T090000", "America/New_York", false},
		{"IANA timezone Europe/London", "20260325T140000", "Europe/London", false},
		{"Windows timezone (Eastern Standard Time)", "20260325T090000", "Eastern Standard Time", true},
		{"Custom VTIMEZONE ID", "20260325T090000", "Custom/Timezone", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop := &ical.Prop{Value: tt.value}
			if tt.tzid != "" {
				prop.Params = ical.Params{"TZID": []string{tt.tzid}}
			}
			_, err := parseDateTime(prop)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test GetEvent for timed event
func TestGetEventTimed(t *testing.T) {
	events := map[string]string{
		"2026-03-25-team-sync.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	result, err := GetEvent(ctx, client, "personal", "2026-03-25-team-sync.ics")
	if err != nil {
		t.Fatalf("GetEvent() error = %v", err)
	}

	if result.CalendarID != "personal" {
		t.Errorf("CalendarID = %v, want 'personal'", result.CalendarID)
	}

	if result.EventID != "2026-03-25-team-sync.ics" {
		t.Errorf("EventID = %v, want '2026-03-25-team-sync.ics'", result.EventID)
	}

	if result.UID != "20260325T090000Z-team-sync@example.com" {
		t.Errorf("UID = %v, want '20260325T090000Z-team-sync@example.com'", result.UID)
	}

	if result.Title != "Team Sync" {
		t.Errorf("Title = %v, want 'Team Sync'", result.Title)
	}

	if result.Description != "Weekly team sync meeting" {
		t.Errorf("Description = %v, want 'Weekly team sync meeting'", result.Description)
	}

	if result.AllDay {
		t.Errorf("AllDay = %v, want false", result.AllDay)
	}
}

// Test GetEvent for all-day event
func TestGetEventAllDay(t *testing.T) {
	events := map[string]string{
		"day-off.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260327-dayoff@example.com
DTSTART;VALUE=DATE:20260327
DTEND;VALUE=DATE:20260328
SUMMARY:Day Off
DESCRIPTION:Vacation day
END:VEVENT
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	result, err := GetEvent(ctx, client, "personal", "day-off.ics")
	if err != nil {
		t.Fatalf("GetEvent() error = %v", err)
	}

	if !result.AllDay {
		t.Errorf("AllDay = %v, want true", result.AllDay)
	}

	// Check that start and end are on the same day for single-day all-day event
	if result.Start.Year() != 2026 || result.Start.Month() != 3 || result.Start.Day() != 27 {
		t.Errorf("Start = %v, want 2026-03-27", result.Start)
	}
}

// Test GetEvent for not found
func TestGetEventNotFound(t *testing.T) {
	events := map[string]string{}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = GetEvent(ctx, client, "personal", "nonexistent.ics")
	if err == nil {
		t.Error("expected error for non-existent event, got nil")
	}
}

// Test GetEvent for invalid calendar ID
func TestGetEventInvalidCalendarID(t *testing.T) {
	events := map[string]string{}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = GetEvent(ctx, client, "nonexistent", "some-event.ics")
	if err == nil {
		t.Error("expected error for invalid calendar ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}
}

// Test GetEvent for malformed iCalendar object
func TestGetEventMalformedObject(t *testing.T) {
	events := map[string]string{
		"malformed.ics": `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
INVALID DATA HERE
END:VCALENDAR`,
	}

	ts := mockEventServer(t, events)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = GetEvent(ctx, client, "personal", "malformed.ics")
	if err == nil {
		t.Error("expected error for malformed iCalendar object, got nil")
	}
}

// mockEventServerForCreate creates a test server that handles PUT requests for event creation.
func mockEventServerForCreate(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && (r.URL.Path == "/dav.php" || r.URL.Path == "/dav.php/"):
			// Current user principal discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<D:current-user-principal>
<D:href>/dav.php/principal/</D:href>
</D:current-user-principal>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/principal/":
			// Calendar home set discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<C:calendar-home-set>
<D:href>/dav.php/calendars/alice/</D:href>
</C:calendar-home-set>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/calendars/alice/":
			// Calendar collection discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:A="http://apple.com/ns/ical/">
<D:response>
<D:href>/dav.php/calendars/alice/personal/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Personal Calendar</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/dav.php/calendars/alice/personal/"):
			// Accept event creation
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.WriteHeader(http.StatusCreated)

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestCreateEventTimed tests creating a timed event.
func TestCreateEventTimed(t *testing.T) {
	ts := mockEventServerForCreate(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	start, _ := time.Parse(time.RFC3339, "2026-03-25T09:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-03-25T09:30:00Z")

	input := CreateEventInput{
		CalendarID:  "personal",
		Title:       "Team Sync",
		Description: "Weekly sync",
		Start:       &start,
		End:         &end,
	}

	result, err := CreateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("CreateEvent() error = %v", err)
	}

	if result.Action != "created" {
		t.Errorf("Action = %v, want 'created'", result.Action)
	}
	if result.CalendarID != "personal" {
		t.Errorf("CalendarID = %v, want 'personal'", result.CalendarID)
	}
	if result.Title != "Team Sync" {
		t.Errorf("Title = %v, want 'Team Sync'", result.Title)
	}
	if result.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if result.UID == "" {
		t.Error("UID should not be empty")
	}
	if !strings.HasSuffix(result.EventID, ".ics") {
		t.Errorf("EventID should end with .ics, got %v", result.EventID)
	}
}

// TestCreateEventAllDay tests creating an all-day event.
func TestCreateEventAllDay(t *testing.T) {
	ts := mockEventServerForCreate(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	date, _ := time.Parse("2006-01-02", "2026-03-27")

	input := CreateEventInput{
		CalendarID:  "personal",
		Title:       "Day Off",
		Description: "Vacation",
		Date:        &date,
	}

	result, err := CreateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("CreateEvent() error = %v", err)
	}

	if result.Action != "created" {
		t.Errorf("Action = %v, want 'created'", result.Action)
	}
	if result.Title != "Day Off" {
		t.Errorf("Title = %v, want 'Day Off'", result.Title)
	}
}

// TestCreateEventValidationError tests validation errors.
func TestCreateEventValidationError(t *testing.T) {
	ts := mockEventServerForCreate(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		input   CreateEventInput
		wantErr string
	}{
		{
			name: "missing calendar_id",
			input: CreateEventInput{
				Title: "Test",
			},
			wantErr: "calendar_id",
		},
		{
			name: "missing title",
			input: CreateEventInput{
				CalendarID: "personal",
			},
			wantErr: "title",
		},
		{
			name: "no date or start/end",
			input: CreateEventInput{
				CalendarID: "personal",
				Title:      "Test",
			},
			wantErr: "either --date",
		},
		{
			name: "both date and start/end",
			input: CreateEventInput{
				CalendarID: "personal",
				Title:      "Test",
				Date:       &time.Time{},
				Start:      &time.Time{},
				End:        &time.Time{},
			},
			wantErr: "cannot use --date with --start/--end",
		},
		{
			name: "start without end",
			input: CreateEventInput{
				CalendarID: "personal",
				Title:      "Test",
				Start:      &time.Time{},
			},
			wantErr: "either --date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateEvent(ctx, client, tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestCreateEventInvalidCalendarID tests creating an event with an invalid calendar ID.
func TestCreateEventInvalidCalendarID(t *testing.T) {
	ts := mockEventServerForCreate(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	start, _ := time.Parse(time.RFC3339, "2026-03-25T09:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-03-25T09:30:00Z")

	input := CreateEventInput{
		CalendarID: "nonexistent",
		Title:      "Test",
		Start:      &start,
		End:        &end,
	}

	_, err = CreateEvent(ctx, client, input)
	if err == nil {
		t.Fatal("expected error for invalid calendar ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should contain 'not found', got: %v", err)
	}
}

// TestGenerateUID tests UID generation.
func TestGenerateUID(t *testing.T) {
	uid1 := generateUID()
	uid2 := generateUID()

	if uid1 == uid2 {
		t.Error("generateUID() should produce unique UIDs")
	}

	if !strings.Contains(uid1, "@caldav-cli") {
		t.Errorf("UID should contain @caldav-cli, got %v", uid1)
	}
}

// TestGenerateEventID tests event filename generation.
func TestGenerateEventID(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Team Sync", "team-sync.ics"},
		{"Day Off", "day-off.ics"},
		{"Meeting", "meeting.ics"},
		{"Multiple   Spaces", "multiple---spaces.ics"},
		{"With_Underscore", "with-underscore.ics"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := generateEventID(tt.title)
			// Check that it ends with the expected sanitized name
			// The date prefix will vary, so we just check the suffix
			if !strings.HasSuffix(got, tt.want) {
				t.Errorf("generateEventID(%q) = %v, want suffix %v", tt.title, got, tt.want)
			}
		})
	}
}

// TestGenerateEventIDEmpty tests event ID generation with empty title.
func TestGenerateEventIDEmpty(t *testing.T) {
	got := generateEventID("")
	if !strings.Contains(got, "event.ics") {
		t.Errorf("generateEventID(\"\") = %v, should contain 'event.ics'", got)
	}
}

// TestGenerateEventIDSpecialChars tests event ID generation with special characters.
func TestGenerateEventIDSpecialChars(t *testing.T) {
	got := generateEventID("Event@#$%^&*()!")
	// Special characters should be removed
	if strings.ContainsAny(got, "@#$%^&*()!") {
		t.Errorf("generateEventID() should remove special chars, got %v", got)
	}
}

// TestGenerateEventIDLongTitle tests event ID generation with very long titles.
func TestGenerateEventIDLongTitle(t *testing.T) {
	longTitle := "This is a very long event title that exceeds the maximum allowed length for event identifiers and should be truncated appropriately"
	got := generateEventID(longTitle)
	// The sanitized title part should be limited
	// Format is: YYYY-MM-DD-<sanitized>.ics
	// So total length should be reasonable
	if len(got) > 100 {
		t.Errorf("generateEventID() should truncate long titles, got length %d: %v", len(got), got)
	}
}

// mockEventServerForUpdate creates a test server that handles GET and PUT requests for event updates.
func mockEventServerForUpdate(t *testing.T, existingEvent string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && (r.URL.Path == "/dav.php" || r.URL.Path == "/dav.php/"):
			// Current user principal discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<D:current-user-principal>
<D:href>/dav.php/principal/</D:href>
</D:current-user-principal>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/principal/":
			// Calendar home set discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<C:calendar-home-set>
<D:href>/dav.php/calendars/alice/</D:href>
</C:calendar-home-set>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/calendars/alice/":
			// Calendar collection discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:A="http://apple.com/ns/ical/">
<D:response>
<D:href>/dav.php/calendars/alice/personal/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Personal Calendar</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "GET" && r.URL.Path == "/dav.php/calendars/alice/personal/team-sync.ics":
			// Return the existing event
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, existingEvent)

		case r.Method == "PUT" && r.URL.Path == "/dav.php/calendars/alice/personal/team-sync.ics":
			// Accept event update
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			w.WriteHeader(http.StatusOK)

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestUpdateEventTitle tests updating an event's title.
func TestUpdateEventTitle(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	newTitle := "Team Sync Updated"

	input := UpdateEventInput{
		CalendarID: "personal",
		EventID:    "team-sync.ics",
		Title:      &newTitle,
	}

	result, err := UpdateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("UpdateEvent() error = %v", err)
	}

	if result.Action != "updated" {
		t.Errorf("Action = %v, want 'updated'", result.Action)
	}
	if result.CalendarID != "personal" {
		t.Errorf("CalendarID = %v, want 'personal'", result.CalendarID)
	}
	if result.EventID != "team-sync.ics" {
		t.Errorf("EventID = %v, want 'team-sync.ics'", result.EventID)
	}
	if result.Title != newTitle {
		t.Errorf("Title = %v, want %v", result.Title, newTitle)
	}
	if result.UID != "20260325T090000Z-team-sync@example.com" {
		t.Errorf("UID should be preserved, got %v", result.UID)
	}
}

// TestUpdateEventDescription tests updating an event's description.
func TestUpdateEventDescription(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	newDescription := "Room 301"

	input := UpdateEventInput{
		CalendarID:  "personal",
		EventID:     "team-sync.ics",
		Description: &newDescription,
	}

	result, err := UpdateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("UpdateEvent() error = %v", err)
	}

	if result.Title != "Team Sync" {
		t.Errorf("Title should be preserved, got %v", result.Title)
	}
	if result.Action != "updated" {
		t.Errorf("Action = %v, want 'updated'", result.Action)
	}
}

// TestUpdateEventTimed tests updating an event's start and end times.
func TestUpdateEventTimed(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	newStart, _ := time.Parse(time.RFC3339, "2026-03-25T10:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2026-03-25T10:30:00Z")

	input := UpdateEventInput{
		CalendarID: "personal",
		EventID:    "team-sync.ics",
		Start:      &newStart,
		End:        &newEnd,
	}

	result, err := UpdateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("UpdateEvent() error = %v", err)
	}

	if result.Action != "updated" {
		t.Errorf("Action = %v, want 'updated'", result.Action)
	}
}

// TestUpdateEventAllDay tests updating a timed event to an all-day event.
func TestUpdateEventAllDay(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	newDate, _ := time.Parse("2006-01-02", "2026-03-27")

	input := UpdateEventInput{
		CalendarID: "personal",
		EventID:    "team-sync.ics",
		Date:       &newDate,
	}

	result, err := UpdateEvent(ctx, client, input)
	if err != nil {
		t.Fatalf("UpdateEvent() error = %v", err)
	}

	if result.Action != "updated" {
		t.Errorf("Action = %v, want 'updated'", result.Action)
	}
}

// TestUpdateEventValidationError tests validation errors for update.
func TestUpdateEventValidationError(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		input   UpdateEventInput
		wantErr string
	}{
		{
			name: "missing calendar_id",
			input: UpdateEventInput{
				EventID: "team-sync.ics",
			},
			wantErr: "calendar_id",
		},
		{
			name: "missing event_id",
			input: UpdateEventInput{
				CalendarID: "personal",
			},
			wantErr: "event_id",
		},
		{
			name: "date with start/end",
			input: UpdateEventInput{
				CalendarID: "personal",
				EventID:    "team-sync.ics",
				Date:       &time.Time{},
				Start:      &time.Time{},
				End:        &time.Time{},
			},
			wantErr: "cannot use --date with --start/--end",
		},
		{
			name: "start without end",
			input: UpdateEventInput{
				CalendarID: "personal",
				EventID:    "team-sync.ics",
				Start:      &time.Time{},
			},
			wantErr: "--end required",
		},
		{
			name: "end without start",
			input: UpdateEventInput{
				CalendarID: "personal",
				EventID:    "team-sync.ics",
				End:        &time.Time{},
			},
			wantErr: "--start required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateEvent(ctx, client, tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestUpdateEventInvalidCalendarID tests updating an event with an invalid calendar ID.
func TestUpdateEventInvalidCalendarID(t *testing.T) {
	existingEvent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Baikal//Baikal//EN
BEGIN:VEVENT
UID:20260325T090000Z-team-sync@example.com
DTSTART:20260325T090000Z
DTEND:20260325T093000Z
SUMMARY:Team Sync
DESCRIPTION:Weekly team sync meeting
END:VEVENT
END:VCALENDAR`

	ts := mockEventServerForUpdate(t, existingEvent)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	newTitle := "Updated"

	input := UpdateEventInput{
		CalendarID: "nonexistent",
		EventID:    "team-sync.ics",
		Title:      &newTitle,
	}

	_, err = UpdateEvent(ctx, client, input)
	if err == nil {
		t.Fatal("expected error for invalid calendar ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should contain 'not found', got: %v", err)
	}
}

// mockEventServerForDelete creates a test server that handles DELETE requests.
func mockEventServerForDelete(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && (r.URL.Path == "/dav.php" || r.URL.Path == "/dav.php/"):
			// Current user principal discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<D:current-user-principal>
<D:href>/dav.php/principal/</D:href>
</D:current-user-principal>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/principal/":
			// Calendar home set discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/principal/</D:href>
<D:propstat>
<D:prop>
<C:calendar-home-set>
<D:href>/dav.php/calendars/alice/</D:href>
</C:calendar-home-set>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/calendars/alice/":
			// Calendar collection discovery
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:A="http://apple.com/ns/ical/">
<D:response>
<D:href>/dav.php/calendars/alice/personal/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Personal Calendar</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "DELETE" && r.URL.Path == "/dav.php/calendars/alice/personal/team-sync.ics":
			// Accept delete request
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestDeleteEventSuccess tests successful event deletion.
func TestDeleteEventSuccess(t *testing.T) {
	ts := mockEventServerForDelete(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	result, err := DeleteEvent(ctx, client, "personal", "team-sync.ics")
	if err != nil {
		t.Fatalf("DeleteEvent() error = %v", err)
	}

	if result.Action != "deleted" {
		t.Errorf("Action = %v, want 'deleted'", result.Action)
	}
	if result.CalendarID != "personal" {
		t.Errorf("CalendarID = %v, want 'personal'", result.CalendarID)
	}
	if result.EventID != "team-sync.ics" {
		t.Errorf("EventID = %v, want 'team-sync.ics'", result.EventID)
	}
}

// TestDeleteEventValidationError tests validation errors for delete.
func TestDeleteEventValidationError(t *testing.T) {
	ts := mockEventServerForDelete(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		calendarID string
		eventID    string
		wantErr    string
	}{
		{
			name:       "missing calendar_id",
			calendarID: "",
			eventID:    "team-sync.ics",
			wantErr:    "calendar_id",
		},
		{
			name:       "missing event_id",
			calendarID: "personal",
			eventID:    "",
			wantErr:    "event_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeleteEvent(ctx, client, tt.calendarID, tt.eventID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestDeleteEventInvalidCalendarID tests deleting an event with an invalid calendar ID.
func TestDeleteEventInvalidCalendarID(t *testing.T) {
	ts := mockEventServerForDelete(t)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	_, err = DeleteEvent(ctx, client, "nonexistent", "team-sync.ics")
	if err == nil {
		t.Fatal("expected error for invalid calendar ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should contain 'not found', got: %v", err)
	}
}
