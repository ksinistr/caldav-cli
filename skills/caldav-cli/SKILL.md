# caldav-cli Skill

A CalDAV CLI skill for managing Baikal calendar server calendars and events.

## Trigger

Invoke this skill when:

- User asks to list, create, read, update, or delete calendar events
- User asks to list or discover calendars
- User mentions "Baikal" or "CalDAV" in the context of calendar management
- User wants to query events by date range

## Description

The caldav-cli tool provides calendar and event management for CalDAV servers. It supports:

- Discovering and listing calendars
- Listing events in a date range
- Creating, reading, updating, and deleting single events
- Timed events and all-day events
- Machine-friendly TOON output for agents
- Human-readable text output

## V1 Scope Limits

Do NOT attempt to:

- Create or modify recurring events
- Create or manage todos (VTODO)
- Add or manage attendees
- Set reminders or alarms

## Configuration

The tool reads configuration from `~/.config/caldav-cli/config.toml`:

```toml
server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
```

Password can also be provided via environment variable:

- `CALDAV_PASSWORD`

## Important Usage Rules

1. **Always use absolute dates** - Do NOT use relative dates like "tomorrow" or "next week". Use ISO 8601 format:

   - Datetimes: `2026-03-25T09:00:00Z`
   - Dates: `2026-03-27`

2. **Use calendar_id for calendar selection** - The `calendar_id` is a stable identifier derived from the calendar path (e.g., `personal`, `work`). Do NOT use full URLs.

3. **Use event_id for event selection** - The `event_id` is the filename including `.ics` extension (e.g., `2026-03-25-team-sync.ics`).

4. **Always use --format toon for machine-readable output** - This ensures consistent, parseable output.

5. **Required flags for events list**: Always provide `--from` and `--to` with ISO 8601 timestamps.

## Common Commands

### List all calendars

```bash
caldav calendars list --format toon
```

### List events in a date range

```bash
caldav events list \
  --calendar-id personal \
  --from 2026-03-24T00:00:00Z \
  --to 2026-03-31T00:00:00Z \
  --format toon
```

### Get event details

```bash
caldav events get \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format json
```

### Create a timed event

```bash
caldav events create \
  --calendar-id personal \
  --title "Team Sync" \
  --start 2026-03-25T09:00:00Z \
  --end 2026-03-25T09:30:00Z \
  --description "Weekly sync" \
  --format json
```

### Create an all-day event

```bash
caldav events create \
  --calendar-id personal \
  --title "Day Off" \
  --date 2026-03-27 \
  --description "Vacation" \
  --format json
```

### Update an event

```bash
caldav events update \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --title "Updated Title" \
  --description "New description" \
  --format json
```

### Delete an event

```bash
caldav events delete \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format text
```

## Output Formats

- `--format toon` - Machine-friendly TOON format (preferred for agents)
- `--format json` - Structured JSON for scripting
- `--format text` - Human-readable text

## Error Handling

Common errors:

- `--from` and `--to` are required for `events list`
- Dates must be in ISO 8601 format
- `--start` and `--end` must both be provided for timed events
- `--date` cannot be used with `--start/--end`
