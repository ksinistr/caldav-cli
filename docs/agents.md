# Agent Usage Guide for caldav-cli

This guide provides recommended calling patterns for AI agents using caldav-cli.

## Core Principles

1. **Always use absolute dates** - ISO 8601 format only
2. **Always use --format toon** for machine-readable output
3. **Use calendar_id and event_id** - never use full URLs
4. **Events list requires --from and --to** - always provide both

## Date Formats

- **Datetime**: `2026-03-25T09:00:00Z` (ISO 8601 with UTC timezone)
- **Date only**: `2026-03-27` (for all-day events)

Never use relative dates like "tomorrow", "next week", etc.

## Identity Contract

- **calendar_id**: Stable identifier derived from calendar path (e.g., `personal`, `work`)
- **event_id**: Filename including `.ics` extension (e.g., `2026-03-25-team-sync.ics`)
- **uid**: Internal iCalendar UID (auto-generated on create)

Full server URLs are intentionally hidden. Always use the CLI identifiers.

## Recommended Calling Patterns

### Discovery

First, discover available calendars:

```bash
caldav calendars list --format toon
```

Example output:
```
calendars:
- calendar_id: personal
  name: Personal Calendar
- calendar_id: work
  name: Work Calendar
```

### List Events

Always specify the date window with absolute timestamps:

```bash
caldav events list \
  --calendar-id personal \
  --from 2026-03-24T00:00:00Z \
  --to 2026-03-31T00:00:00Z \
  --format toon
```

### Get Event Details

Fetch full event details:

```bash
caldav events get \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format toon
```

### Create Timed Event

Use `--start` and `--end` for timed events:

```bash
caldav events create \
  --calendar-id personal \
  --title "Team Sync" \
  --start 2026-03-25T09:00:00Z \
  --end 2026-03-25T09:30:00Z \
  --description "Weekly sync" \
  --format toon
```

### Create All-Day Event

Use `--date` for all-day events:

```bash
caldav events create \
  --calendar-id personal \
  --title "Day Off" \
  --date 2026-03-27 \
  --description "Vacation" \
  --format toon
```

### Update Event

Only include flags for fields you want to change. Omitted flags preserve existing values.

```bash
# Update title only
caldav events update \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --title "New Title" \
  --format toon

# Update description and time
caldav events update \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --description "Room 301" \
  --start 2026-03-25T10:00:00Z \
  --end 2026-03-25T10:30:00Z \
  --format toon
```

### Delete Event

```bash
caldav events delete \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format toon
```

## Output Format Preference

- **Default to `--format toon`** for consistent machine-readable output
- Use `--format json` when you need standard JSON for tools like `jq`
- Use `--format text` only when the user explicitly wants human-readable output

## Error Handling

Common error responses and their meanings:

| Error | Cause | Solution |
|-------|-------|----------|
| `server_url is required` | Config missing or incomplete | Create `~/.config/caldav-cli/config.toml` |
| `invalid --from date` | Wrong date format | Use ISO 8601: `2026-03-24T00:00:00Z` |
| `--to must be after --from` | Inverted date range | Ensure `--to` is later than `--from` |
| `--end required when --start is provided` | Incomplete time range | Provide both `--start` and `--end` |
| `cannot use --date with --start/--end` | Mixed event types | Use `--date` for all-day OR `--start/--end` for timed |
| `failed to discover calendars` | Server connection or auth | Check server URL and credentials |

## V1 Scope Limits

Do NOT attempt:
- Recurring events (RRULE)
- Todos (VTODO)
- Attendees
- Reminders/alarms
- Multiple profiles (only one profile in v1)

## Complete Example Workflow

```bash
# 1. Discover calendars
caldav calendars list --format toon

# 2. List events for a week
caldav events list \
  --calendar-id personal \
  --from 2026-03-24T00:00:00Z \
  --to 2026-03-31T23:59:59Z \
  --format toon

# 3. Create a new event
caldav events create \
  --calendar-id personal \
  --title "Product Review" \
  --start 2026-03-26T14:00:00Z \
  --end 2026-03-26T15:00:00Z \
  --description "Q1 product review meeting" \
  --format toon

# 4. Get the created event (event_id from create response)
caldav events get \
  --calendar-id personal \
  --event-id 20260326T140000Z-product-review.ics \
  --format toon

# 5. Update the event
caldav events update \
  --calendar-id personal \
  --event-id 20260326T140000Z-product-review.ics \
  --description "Q1 product review - Room 201" \
  --format toon

# 6. Delete the event
caldav events delete \
  --calendar-id personal \
  --event-id 20260326T140000Z-product-review.ics \
  --format toon
```

## Configuration for Agents

Ensure the environment has:
1. Config file at `~/.config/caldav-cli/config.toml`
2. Password via `CALDAV_PASSWORD` environment variable (recommended)
3. `caldav-cli` binary in PATH

Example config:
```toml
server_url = "https://baikal.example.com/dav.php/"
username = "alice"
insecure_skip_verify = false
# Password via CALDAV_PASSWORD
```
