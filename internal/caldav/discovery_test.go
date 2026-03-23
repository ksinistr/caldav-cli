package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/emersion/go-webdav/caldav"
	"github.com/ksinistr/caldav-cli/internal/config"
)

// mockCalDAVServer creates a test server that mimics Baikal's CalDAV responses.
func mockCalDAVServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
<D:response>
<D:href>/dav.php/calendars/alice/work/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Work Calendar</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
<D:response>
<D:href>/dav.php/calendars/alice/team-projects/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
<D:displayname>Team Projects</D:displayname>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		case r.Method == "PROPFIND" && strings.HasPrefix(r.URL.Path, "/dav.php/calendars/alice/"):
			// Individual calendar props (not used in discovery flow)
			w.WriteHeader(http.StatusMultiStatus)

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	u, _ := url.Parse(ts.URL)
	cfg := &config.Config{
		ServerURL:          u.String() + "/dav.php/",
		Username:           "alice",
		Password:           "testpass",
		InsecureSkipVerify: true,
	}

	return ts, cfg
}

func TestDiscoverCalendars(t *testing.T) {
	ts, cfg := mockCalDAVServer(t)
	defer ts.Close()

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	result, err := DiscoverCalendars(ctx, client)
	if err != nil {
		t.Fatalf("DiscoverCalendars() error = %v", err)
	}

	if len(result.Calendars) != 3 {
		t.Errorf("DiscoverCalendars() returned %d calendars, want 3", len(result.Calendars))
	}

	// Check first calendar
	if result.Calendars[0].CalendarID != "personal" {
		t.Errorf("Calendar[0].CalendarID = %v, want 'personal'", result.Calendars[0].CalendarID)
	}
	if result.Calendars[0].Name != "Personal Calendar" {
		t.Errorf("Calendar[0].Name = %v, want 'Personal Calendar'", result.Calendars[0].Name)
	}

	// Check second calendar
	if result.Calendars[1].CalendarID != "work" {
		t.Errorf("Calendar[1].CalendarID = %v, want 'work'", result.Calendars[1].CalendarID)
	}
	if result.Calendars[1].Name != "Work Calendar" {
		t.Errorf("Calendar[1].Name = %v, want 'Work Calendar'", result.Calendars[1].Name)
	}

	// Check third calendar
	if result.Calendars[2].CalendarID != "team-projects" {
		t.Errorf("Calendar[2].CalendarID = %v, want 'team-projects'", result.Calendars[2].CalendarID)
	}
	if result.Calendars[2].Name != "Team Projects" {
		t.Errorf("Calendar[2].Name = %v, want 'Team Projects'", result.Calendars[2].Name)
	}
}

func TestDiscoverCalendarsWithEmptyDisplayName(t *testing.T) {
	// Server that returns calendars without display names
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && (r.URL.Path == "/dav.php" || r.URL.Path == "/dav.php/"):
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
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			// Note: no displayname elements
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/dav.php/calendars/alice/personal/</D:href>
<D:propstat>
<D:prop>
<D:resourcetype>
<D:collection/>
<C:calendar/>
</D:resourcetype>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
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
	result, err := DiscoverCalendars(ctx, client)
	if err != nil {
		t.Fatalf("DiscoverCalendars() error = %v", err)
	}

	if len(result.Calendars) != 1 {
		t.Errorf("DiscoverCalendars() returned %d calendars, want 1", len(result.Calendars))
	}

	// When display name is empty, it should fall back to calendar_id
	if result.Calendars[0].Name != "personal" {
		t.Errorf("Calendar[0].Name = %v, want 'personal' (fallback to calendar_id)", result.Calendars[0].Name)
	}
}

func TestResolveCalendarByPath(t *testing.T) {
	calendars := []caldav.Calendar{
		{Path: "/dav.php/calendars/alice/personal/"},
		{Path: "/dav.php/calendars/alice/work/"},
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "find existing calendar",
			path:    "/dav.php/calendars/alice/personal/",
			want:    "/dav.php/calendars/alice/personal/",
			wantErr: false,
		},
		{
			name:    "calendar not found",
			path:    "/dav.php/calendars/alice/nonexistent/",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCalendarByPath(tt.path, calendars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCalendarByPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Path != tt.want {
				t.Errorf("ResolveCalendarByPath() = %v, want %v", got.Path, tt.want)
			}
		})
	}
}

func TestNormalizeHref(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			name: "already normalized",
			href: "/dav.php/calendars/alice/",
			want: "/dav.php/calendars/alice/",
		},
		{
			name: "missing leading slash",
			href: "dav.php/calendars/alice/",
			want: "/dav.php/calendars/alice/",
		},
		{
			name: "with whitespace",
			href: "  /dav.php/calendars/alice/  ",
			want: "/dav.php/calendars/alice/",
		},
		{
			name: "double slashes",
			href: "//dav.php//calendars//alice//",
			want: "/dav.php/calendars/alice/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeHref(tt.href); got != tt.want {
				t.Errorf("NormalizeHref() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPrincipalURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "principal with /principal/",
			url:  "/dav.php/principal/",
			want: true,
		},
		{
			name: "principal with /user/",
			url:  "/dav.php/user/alice/",
			want: true,
		},
		{
			name: "calendar path",
			url:  "/dav.php/calendars/alice/personal/",
			want: false,
		},
		{
			name: "no calendars keyword",
			url:  "/dav.php/home/",
			want: true,
		},
		{
			name: "root",
			url:  "/dav.php/",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrincipalURL(tt.url); got != tt.want {
				t.Errorf("IsPrincipalURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
