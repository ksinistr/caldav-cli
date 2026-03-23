package caldav

import (
	"testing"

	"github.com/emersion/go-webdav/caldav"
)

func TestCalendarIDFromHref(t *testing.T) {
	tests := []struct {
		name    string
		href    string
		want    string
		wantErr bool
	}{
		{
			name:    "simple personal calendar",
			href:    "/dav.php/calendars/alice/personal/",
			want:    "personal",
			wantErr: false,
		},
		{
			name:    "work calendar without trailing slash",
			href:    "/dav.php/calendars/alice/work",
			want:    "work",
			wantErr: false,
		},
		{
			name:    "calendar with numeric path",
			href:    "/dav.php/calendars/bob/calendar42/",
			want:    "calendar42",
			wantErr: false,
		},
		{
			name:    "calendar with dashes",
			href:    "/dav.php/calendars/alice/team-calendar/",
			want:    "team-calendar",
			wantErr: false,
		},
		{
			name:    "root level href - returns username",
			href:    "/dav.php/calendars/alice/",
			want:    "alice",
			wantErr: false,
		},
		{
			name:    "empty href - error",
			href:    "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "just slash - error",
			href:    "/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "deeply nested calendar",
			href:    "/dav.php/calendars/alice/work/projects/",
			want:    "projects",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalendarIDFromHref(tt.href)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalendarIDFromHref() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CalendarIDFromHref() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckDuplicateIDs(t *testing.T) {
	tests := []struct {
		name           string
		calendars      []caldav.Calendar
		wantDuplicates map[string]int
	}{
		{
			name: "no duplicates",
			calendars: []caldav.Calendar{
				{Path: "/dav.php/calendars/alice/personal/"},
				{Path: "/dav.php/calendars/alice/work/"},
			},
			wantDuplicates: map[string]int{
				"personal": 1,
				"work":     1,
			},
		},
		{
			name: "with duplicates",
			calendars: []caldav.Calendar{
				{Path: "/dav.php/calendars/alice/personal/"},
				{Path: "/dav.php/calendars/alice/work/"},
				{Path: "/dav.php/calendars/alice/personal2/"},
			},
			wantDuplicates: map[string]int{
				"personal":  1,
				"work":      1,
				"personal2": 1,
			},
		},
		{
			name: "actual duplicate paths",
			calendars: []caldav.Calendar{
				{Path: "/dav.php/calendars/alice/cal/"},
				{Path: "/dav.php/calendars/alice/work/"},
				{Path: "/dav.php/calendars/alice/cal/"},
			},
			wantDuplicates: map[string]int{
				"cal":  2,
				"work": 1,
			},
		},
		{
			name:           "empty calendars",
			calendars:      []caldav.Calendar{},
			wantDuplicates: map[string]int{
				// empty map
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckDuplicateIDs(tt.calendars)
			if len(got) != len(tt.wantDuplicates) {
				t.Errorf("CheckDuplicateIDs() length = %v, want %v", len(got), len(tt.wantDuplicates))
			}
			for id, count := range tt.wantDuplicates {
				if got[id] != count {
					t.Errorf("CheckDuplicateIDs()[%v] = %v, want %v", id, got[id], count)
				}
			}
		})
	}
}

func TestResolveCalendarID(t *testing.T) {
	calendars := []caldav.Calendar{
		{Path: "/dav.php/calendars/alice/personal/"},
		{Path: "/dav.php/calendars/alice/work/"},
		{Path: "/dav.php/calendars/alice/team-projects/"},
	}
	tests := []struct {
		name       string
		calendarID string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "find personal",
			calendarID: "personal",
			wantPath:   "/dav.php/calendars/alice/personal/",
			wantErr:    false,
		},
		{
			name:       "find work",
			calendarID: "work",
			wantPath:   "/dav.php/calendars/alice/work/",
			wantErr:    false,
		},
		{
			name:       "find with dash",
			calendarID: "team-projects",
			wantPath:   "/dav.php/calendars/alice/team-projects/",
			wantErr:    false,
		},
		{
			name:       "not found",
			calendarID: "nonexistent",
			wantPath:   "",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCalendarID(tt.calendarID, calendars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCalendarID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Path != tt.wantPath {
				t.Errorf("ResolveCalendarID() path = %v, want %v", got.Path, tt.wantPath)
			}
		})
	}
}

func TestCalendarIDError(t *testing.T) {
	tests := []struct {
		name    string
		href    string
		reason  string
		wantMsg string
	}{
		{
			name:    "with href",
			href:    "/dav.php/calendars/alice/",
			reason:  "empty calendar path component",
			wantMsg: "cannot derive calendar_id from href '/dav.php/calendars/alice/': empty calendar path component",
		},
		{
			name:    "without href",
			href:    "",
			reason:  "test error",
			wantMsg: "cannot derive calendar_id: test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &CalendarIDError{Href: tt.href, Reason: tt.reason}
			if msg := err.Error(); msg != tt.wantMsg {
				t.Errorf("CalendarIDError.Error() = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}

func TestCalendarNotFoundError(t *testing.T) {
	err := &CalendarNotFoundError{CalendarID: "personal"}
	wantMsg := "calendar with id 'personal' not found"
	if msg := err.Error(); msg != wantMsg {
		t.Errorf("CalendarNotFoundError.Error() = %v, want %v", msg, wantMsg)
	}
}
