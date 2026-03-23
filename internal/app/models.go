package app

import "time"

// Calendar represents a discovered calendar with its CLI-facing identifier.
type Calendar struct {
	CalendarID string `json:"calendar_id" toon:"calendar_id"`
	Name       string `json:"name" toon:"name"`
}

// CalendarsList is the result of listing all calendars.
type CalendarsList struct {
	Calendars []Calendar `json:"calendars" toon:"calendars"`
}

// Event represents a single event (timed or all-day) with its identifiers.
type Event struct {
	CalendarID  string    `json:"calendar_id" toon:"calendar_id"`
	EventID     string    `json:"event_id" toon:"event_id"`
	UID         string    `json:"uid" toon:"uid"`
	Title       string    `json:"title" toon:"title"`
	Description string    `json:"description,omitempty" toon:"description,omitempty"`
	Start       time.Time `json:"start" toon:"start"`
	End         time.Time `json:"end" toon:"end"`
	AllDay      bool      `json:"all_day" toon:"all_day"`
}

// EventsList is the result of listing events in a calendar.
type EventsList struct {
	CalendarID string  `json:"calendar_id" toon:"calendar_id"`
	From       string  `json:"from" toon:"from"`
	To         string  `json:"to" toon:"to"`
	Events     []Event `json:"events" toon:"events"`
}

// EventDetail is the result of fetching a single event.
type EventDetail struct {
	Event
}

// MutationResult is the result of create, update, or delete operations.
type MutationResult struct {
	Action     string `json:"action" toon:"action"`
	CalendarID string `json:"calendar_id" toon:"calendar_id"`
	EventID    string `json:"event_id,omitempty" toon:"event_id,omitempty"`
	UID        string `json:"uid,omitempty" toon:"uid,omitempty"`
	Title      string `json:"title,omitempty" toon:"title,omitempty"`
	Message    string `json:"message" toon:"message"`
}

// Error wraps an error with optional context for structured output.
type Error struct {
	Message string `json:"error" toon:"error"`
	Code    string `json:"code,omitempty" toon:"code,omitempty"`
}
