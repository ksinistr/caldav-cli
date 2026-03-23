# Installing caldav-cli

This document provides complete installation instructions for caldav-cli. Hand this URL to another coding agent to enable automatic installation:

https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.md

## Quick Install

Install the latest release to `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | bash
```

Add `~/.local/bin` to your PATH if it's not already there:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Custom Install Directory

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

## Version-Pinned Install

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | VERSION=v0.1.0 bash
```

## Custom Repository

Install from a custom repository:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | REPO_SLUG=custom/repo bash
```

## Verification

Verify the installation:

```bash
caldav --help
caldav calendars list
```

## Configuration

From a local checkout, create the config directory and file:

```bash
mkdir -p ~/.config/caldav-cli
cp config.example.toml ~/.config/caldav-cli/config.toml
```

If you are not working from a local checkout, create `~/.config/caldav-cli/config.toml` with:

```toml
server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
insecure_skip_verify = false
```

Or set your password via environment variable:

```bash
export CALDAV_PASSWORD="your-password-here"
```

## Installing as a Skill for AI Agents

For Claude Code or other agent systems, install the skill definition:

```bash
curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/skills/caldav-cli/install.sh | bash -s -- /path/to/skills
```

Or from a local checkout:

```bash
bash skills/caldav-cli/install.sh /path/to/skills
```

## Manual Install

Download the binary for your platform from GitHub Releases:

https://github.com/ksinistr/caldav-cli/releases

```bash
# Download the binary
wget https://github.com/ksinistr/caldav-cli/releases/download/v0.1.0/caldav-cli-linux-amd64 -O caldav-cli

# Make it executable
chmod +x caldav-cli

# Move to your PATH
mv caldav-cli ~/.local/bin/
```

## Building from Source

Requirements: Go 1.26.1 or later

```bash
git clone https://github.com/ksinistr/caldav-cli.git
cd caldav-cli
make build
mv caldav-cli ~/.local/bin/
```

## Next Steps

After installation:

1. Configure your server credentials in `~/.config/caldav-cli/config.toml`
2. List calendars: `caldav calendars list --format toon`
3. See the [README](https://github.com/ksinistr/caldav-cli) for usage examples
4. See [docs/agents.md](https://raw.githubusercontent.com/ksinistr/caldav-cli/main/docs/agents.md) for agent usage patterns
