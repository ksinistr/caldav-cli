package out

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ksinistr/caldav-cli/internal/app"
)

// TestParseFormat tests the ParseFormat function with various inputs.
func TestParseFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Format
	}{
		{
			name:  "lowercase json",
			input: "json",
			want:  FormatJSON,
		},
		{
			name:  "uppercase JSON",
			input: "JSON",
			want:  FormatJSON,
		},
		{
			name:  "mixed case Json",
			input: "Json",
			want:  FormatJSON,
		},
		{
			name:  "lowercase toon",
			input: "toon",
			want:  FormatTOON,
		},
		{
			name:  "uppercase TOON",
			input: "TOON",
			want:  FormatTOON,
		},
		{
			name:  "mixed case Toon",
			input: "Toon",
			want:  FormatTOON,
		},
		{
			name:  "lowercase text",
			input: "text",
			want:  FormatText,
		},
		{
			name:  "uppercase TEXT",
			input: "TEXT",
			want:  FormatText,
		},
		{
			name:  "mixed case Text",
			input: "Text",
			want:  FormatText,
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "invalid format returns empty",
			input: "yaml",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFormat(tt.input)
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestFormatResolver_Resolve tests format resolution with various configurations.
func TestFormatResolver_Resolve(t *testing.T) {
	tests := []struct {
		name        string
		explicit    string
		wantNonTTY  Format
		description string
	}{
		{
			name:        "explicit json overrides TTY",
			explicit:    "json",
			wantNonTTY:  FormatJSON,
			description: "explicit json should always give json",
		},
		{
			name:        "explicit toon overrides TTY",
			explicit:    "toon",
			wantNonTTY:  FormatTOON,
			description: "explicit toon should always give toon",
		},
		{
			name:        "explicit text overrides TTY",
			explicit:    "text",
			wantNonTTY:  FormatText,
			description: "explicit text should always give text",
		},
		{
			name:        "empty explicit uses default",
			explicit:    "",
			wantNonTTY:  FormatTOON,
			description: "no explicit format should default to TOON for non-TTY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr := NewFormatResolver(tt.explicit)
			got := fr.Resolve()
			if got != tt.wantNonTTY {
				t.Errorf("FormatResolver.Resolve() with explicit=%q, IsTTY=false = %q, want %q (%s)",
					tt.explicit, got, tt.wantNonTTY, tt.description)
			}
		})
	}
}

// TestJSONRenderer tests JSON output rendering.
func TestJSONRenderer(t *testing.T) {
	ctx := context.Background()
	renderer := &jsonRenderer{}

	t.Run("render calendars list", func(t *testing.T) {
		data := &app.CalendarsList{
			Calendars: []app.Calendar{
				{CalendarID: "personal", Name: "Personal"},
				{CalendarID: "work", Name: "Work Calendar"},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		// Verify JSON structure
		if !strings.Contains(output, `"calendar_id"`) {
			t.Errorf("JSON output missing calendar_id field: %s", output)
		}
		if !strings.Contains(output, `"personal"`) {
			t.Errorf("JSON output missing calendar ID 'personal': %s", output)
		}
	})

	t.Run("render events list", func(t *testing.T) {
		data := &app.EventsList{
			CalendarID: "personal",
			From:       "2026-03-24T00:00:00Z",
			To:         "2026-03-31T00:00:00Z",
			Events: []app.Event{
				{
					CalendarID:  "personal",
					EventID:     "meeting.ics",
					UID:         "meeting-uid@example.com",
					Title:       "Team Meeting",
					Description: "Weekly sync",
				},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, `"event_id"`) {
			t.Errorf("JSON output missing event_id field: %s", output)
		}
	})

	t.Run("render mutation result", func(t *testing.T) {
		data := &app.MutationResult{
			Action:     "created",
			CalendarID: "personal",
			EventID:    "new-event.ics",
			UID:        "new-uid@example.com",
			Title:      "New Event",
			Message:    "Event created successfully",
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, `"action"`) {
			t.Errorf("JSON output missing action field: %s", output)
		}
	})

	t.Run("render error envelope", func(t *testing.T) {
		data := ErrorEnvelope{Error: "test error message"}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, `"error"`) {
			t.Errorf("JSON output missing error field: %s", output)
		}
		if !strings.Contains(output, "test error message") {
			t.Errorf("JSON output missing error message: %s", output)
		}
	})
}

// TestToonRenderer tests TOON output rendering.
func TestToonRenderer(t *testing.T) {
	ctx := context.Background()
	renderer := &toonRenderer{}

	t.Run("render calendars list", func(t *testing.T) {
		data := &app.CalendarsList{
			Calendars: []app.Calendar{
				{CalendarID: "personal", Name: "Personal"},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		// TOON uses underscores for keys
		if !strings.Contains(output, "calendar_id") {
			t.Errorf("TOON output missing calendar_id: %s", output)
		}
	})

	t.Run("render events list", func(t *testing.T) {
		data := &app.EventsList{
			CalendarID: "personal",
			From:       "2026-03-24T00:00:00Z",
			To:         "2026-03-31T00:00:00Z",
			Events: []app.Event{
				{
					CalendarID: "personal",
					EventID:    "event.ics",
					Title:      "Test Event",
				},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		// Verify TOON format
		if !strings.Contains(output, "event_id") {
			t.Errorf("TOON output missing event_id: %s", output)
		}
	})

	t.Run("render mutation result", func(t *testing.T) {
		data := &app.MutationResult{
			Action:  "created",
			Message: "Event created",
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "action") {
			t.Errorf("TOON output missing action: %s", output)
		}
	})

	t.Run("render error envelope", func(t *testing.T) {
		data := ErrorEnvelope{Error: "test error"}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "error") {
			t.Errorf("TOON output missing error field: %s", output)
		}
	})
}

// TestTextRenderer tests human-readable text output.
func TestTextRenderer(t *testing.T) {
	ctx := context.Background()
	renderer := &textRenderer{}

	t.Run("render calendars list", func(t *testing.T) {
		data := &app.CalendarsList{
			Calendars: []app.Calendar{
				{CalendarID: "personal", Name: "Personal Calendar"},
				{CalendarID: "work", Name: "Work"},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		// Verify human-readable format
		if !strings.Contains(output, "calendar(s)") {
			t.Errorf("Text output missing calendar count: %s", output)
		}
		if !strings.Contains(output, "Personal Calendar") {
			t.Errorf("Text output missing calendar name: %s", output)
		}
		if !strings.Contains(output, "id: personal") {
			t.Errorf("Text output missing calendar ID: %s", output)
		}
	})

	t.Run("render empty calendars list", func(t *testing.T) {
		data := &app.CalendarsList{
			Calendars: []app.Calendar{},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "No calendars found") {
			t.Errorf("Text output missing 'no calendars' message: %s", output)
		}
	})

	t.Run("render events list", func(t *testing.T) {
		data := &app.EventsList{
			CalendarID: "personal",
			From:       "2026-03-24T00:00:00Z",
			To:         "2026-03-31T00:00:00Z",
			Events: []app.Event{
				{
					CalendarID:  "personal",
					EventID:     "meeting.ics",
					Title:       "Team Meeting",
					Description: "Weekly sync",
				},
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Events in") {
			t.Errorf("Text output missing events header: %s", output)
		}
		if !strings.Contains(output, "Team Meeting") {
			t.Errorf("Text output missing event title: %s", output)
		}
	})

	t.Run("render empty events list", func(t *testing.T) {
		data := &app.EventsList{
			CalendarID: "personal",
			From:       "2026-03-24T00:00:00Z",
			To:         "2026-03-31T00:00:00Z",
			Events:     []app.Event{},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "No events found") {
			t.Errorf("Text output missing 'no events' message: %s", output)
		}
	})

	t.Run("render event detail", func(t *testing.T) {
		data := &app.EventDetail{
			Event: app.Event{
				CalendarID:  "personal",
				EventID:     "meeting.ics",
				UID:         "uid@example.com",
				Title:       "Important Meeting",
				Description: "Discuss project status",
			},
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Event: Important Meeting") {
			t.Errorf("Text output missing event title: %s", output)
		}
	})

	t.Run("render mutation result", func(t *testing.T) {
		data := &app.MutationResult{
			Action:     "created",
			CalendarID: "personal",
			EventID:    "new-event.ics",
			UID:        "new-uid@example.com",
			Title:      "New Event",
			Message:    "Event created successfully",
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "created: Event created successfully") {
			t.Errorf("Text output missing action and message: %s", output)
		}
	})

	t.Run("render error envelope", func(t *testing.T) {
		data := ErrorEnvelope{Error: "something went wrong"}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Error: something went wrong") {
			t.Errorf("Text output missing error message: %s", output)
		}
	})

	t.Run("render app error with code", func(t *testing.T) {
		data := &app.Error{
			Message: "authentication failed",
			Code:    "AUTH_ERROR",
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Error [AUTH_ERROR]: authentication failed") {
			t.Errorf("Text output incorrect format: %s", output)
		}
	})

	t.Run("render app error without code", func(t *testing.T) {
		data := &app.Error{
			Message: "generic error",
		}
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, data)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Error: generic error") {
			t.Errorf("Text output incorrect format: %s", output)
		}
	})
}

// TestNewRenderer tests the renderer factory function.
func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   string
	}{
		{
			name:   "json format",
			format: FormatJSON,
			want:   "*jsonRenderer",
		},
		{
			name:   "toon format",
			format: FormatTOON,
			want:   "*toonRenderer",
		},
		{
			name:   "text format",
			format: FormatText,
			want:   "*textRenderer",
		},
		{
			name:   "unknown format defaults to json",
			format: Format("unknown"),
			want:   "*jsonRenderer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRenderer(tt.format)
			typeStr := ""
			switch got.(type) {
			case *jsonRenderer:
				typeStr = "*jsonRenderer"
			case *toonRenderer:
				typeStr = "*toonRenderer"
			case *textRenderer:
				typeStr = "*textRenderer"
			}
			if typeStr != tt.want {
				t.Errorf("NewRenderer(%q) = %v, want %s", tt.format, typeStr, tt.want)
			}
		})
	}
}
