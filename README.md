# caldav-cli

An agent-first CalDAV CLI for Baikal calendar server. Manage calendars and events from your terminal or from AI agents.

## Features

- Single profile, multiple calendars
- Discover and list calendars automatically
- Create, read, update, and delete single events
- Timed events and all-day events
- Machine-friendly TOON output for AI agents
- Human-friendly text output for interactive use
- JSON output for scripting and pipelines

## V1 Scope Limits

The following features are intentionally not supported in v1:
- Recurring events
- Todos (VTODO)
- Attendees
- Reminders/alarms

## Installation

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | bash
```

This installs `caldav-cli` to `~/.local/bin` by default.

### Custom Install Directory

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

### Version-Pinned Install

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | VERSION=v0.1.0 bash
```

### Verify Installation

```bash
caldav --help
```

### Install as a Skill for AI Agents

For Claude Code or other agent systems:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/skills/caldav-cli/install.sh | bash -s -- /path/to/skills
```

Or from a local checkout:

```bash
bash skills/caldav-cli/install.sh /path/to/skills
```

## Configuration

The repo includes an example config at `config.example.toml`.

Create your local config from it:

```bash
mkdir -p ~/.config/caldav-cli
cp config.example.toml ~/.config/caldav-cli/config.toml
```

Then edit `~/.config/caldav-cli/config.toml`:

```toml
server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
insecure_skip_verify = false
```

### Password Sources

The password can be provided in three ways (in order of priority):

1. Environment variable `CALDAV_PASSWORD`
2. Environment variable `BAIKAL_PASSWORD` (fallback)
3. Config file `password` field

Example:

```bash
export CALDAV_PASSWORD="correct-horse-battery-staple"
caldav calendars list
```

### Config File Override

Use a custom config path:

```bash
caldav --config /path/to/config.toml calendars list
```

## Output Formats

The CLI supports three output formats:

- `toon` - Machine-friendly TOON format (default for non-TTY)
- `text` - Human-readable text (default for TTY)
- `json` - Structured JSON for scripting

The format can be explicitly set with `--format`:

```bash
caldav --format toon calendars list
caldav --format text calendars list
caldav --format json calendars list
```

## Calendar and Event Identity

- `calendar_id` - A stable identifier derived from the discovered calendar path (e.g., `personal`, `work`)
- `event_id` - The event filename including `.ics` extension (e.g., `2026-03-25-team-sync.ics`)

Full server URLs are intentionally hidden from the CLI surface. Use `calendar_id` and `event_id` for all operations.

## Usage

### List Calendars

```bash
caldav calendars list --format toon
```

### List Events

List events in a date range (required `--from` and `--to`):

```bash
caldav events list \
  --calendar-id personal \
  --from 2026-03-24T00:00:00Z \
  --to 2026-03-31T00:00:00Z \
  --format toon
```

### Get Event Details

```bash
caldav events get \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format json
```

### Create a Timed Event

```bash
caldav events create \
  --calendar-id personal \
  --title "Team Sync" \
  --start 2026-03-25T09:00:00Z \
  --end 2026-03-25T09:30:00Z \
  --description "Weekly sync" \
  --format json
```

### Create an All-Day Event

```bash
caldav events create \
  --calendar-id personal \
  --title "Day Off" \
  --date 2026-03-27 \
  --description "Vacation" \
  --format json
```

### Update an Event

Omitted flags preserve existing values:

```bash
caldav events update \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --title "Team Sync Updated" \
  --description "Room 301" \
  --format json
```

### Delete an Event

```bash
caldav events delete \
  --calendar-id personal \
  --event-id 2026-03-25-team-sync.ics \
  --format text
```

## Development

### Requirements

- Go 1.26.1 or later

### Build

```bash
make build
```

### Test

```bash
make test
```

## License

MIT

## Links

- GitHub: https://github.com/ksinistr/caldav-cli
- Install Docs: https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.md
