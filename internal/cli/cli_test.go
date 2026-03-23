package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/ksinistr/caldav-cli/internal/out"
	"github.com/urfave/cli/v3"
)

func TestNewCommandLine_HasCorrectName(t *testing.T) {
	cmd := New()

	if cmd.Name != "caldav" {
		t.Errorf("expected name 'caldav', got '%s'", cmd.Name)
	}
}

func TestNewCommandLine_HasGlobalFlags(t *testing.T) {
	cmd := New()

	var flagNames []string
	for _, f := range cmd.Flags {
		flagNames = append(flagNames, f.Names()[0])
	}

	foundConfig := false
	foundFormat := false
	for _, name := range flagNames {
		if name == "config" {
			foundConfig = true
		}
		if name == "format" {
			foundFormat = true
		}
	}

	if !foundConfig {
		t.Error("expected 'config' flag not found")
	}
	if !foundFormat {
		t.Error("expected 'format' flag not found")
	}
}

func TestNewCommandLine_HasCalendarsCommand(t *testing.T) {
	cmd := New()

	found := false
	for _, c := range cmd.Commands {
		if c.Name == "calendars" {
			found = true
			break
		}
	}

	if !found {
		t.Error("missing 'calendars' subcommand")
	}
}

func TestNewCommandLine_HasEventsCommand(t *testing.T) {
	cmd := New()

	found := false
	for _, c := range cmd.Commands {
		if c.Name == "events" {
			found = true
			break
		}
	}

	if !found {
		t.Error("missing 'events' subcommand")
	}
}

func TestParseFormat_Functions(t *testing.T) {
	tests := []struct {
		input    string
		expected out.Format
	}{
		{"json", out.FormatJSON},
		{"toon", out.FormatTOON},
		{"text", out.FormatText},
		{"JSON", out.FormatJSON},
		{"TOON", out.FormatTOON},
		{"TEXT", out.FormatText},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := out.ParseFormat(tt.input)
		if result != tt.expected {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestFormatResolver_Resolve(t *testing.T) {
	tests := []struct {
		name     string
		resolver *out.FormatResolver
		expected out.Format
	}{
		{"explicit json", &out.FormatResolver{ExplicitFormat: out.FormatJSON}, out.FormatJSON},
		{"explicit toon", &out.FormatResolver{ExplicitFormat: out.FormatTOON}, out.FormatTOON},
		{"explicit text", &out.FormatResolver{ExplicitFormat: out.FormatText}, out.FormatText},
		{"no explicit format", &out.FormatResolver{}, out.FormatTOON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.resolver.Resolve()
			if result != tt.expected {
				t.Errorf("Resolve() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetFormatResolver(t *testing.T) {
	// Create a minimal command for testing
	cmd := &cli.Command{}
	// This just verifies the function exists and returns non-nil
	res := GetFormatResolver(cmd)
	if res == nil {
		t.Error("GetFormatResolver returned nil")
	}
}

func TestCLI_Integration_HasSubcommands(t *testing.T) {
	cmd := New()

	subcommandNames := make(map[string]bool)
	for _, c := range cmd.Commands {
		subcommandNames[c.Name] = true
		for _, sc := range c.Commands {
			subcommandNames[sc.Name] = true
		}
	}

	expectedSubcommands := []string{"calendars", "events", "list", "get", "create", "update", "delete"}
	for _, expected := range expectedSubcommands {
		if !subcommandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestRun_ParsesFlags(t *testing.T) {
	cmd := New()

	// Verify command structure
	if cmd.Name != "caldav" {
		t.Errorf("expected name 'caldav', got '%s'", cmd.Name)
	}

	// Check that events list has the required flags
	eventsListFound := false
	for _, c := range cmd.Commands {
		if c.Name == "events" {
			for _, sc := range c.Commands {
				if sc.Name == "list" {
					eventsListFound = true
					foundCalendarID := false
					foundFrom := false
					foundTo := false
					for _, f := range sc.Flags {
						names := f.Names()
						for _, name := range names {
							if name == "calendar-id" {
								foundCalendarID = true
							}
							if name == "from" {
								foundFrom = true
							}
							if name == "to" {
								foundTo = true
							}
						}
					}
					if !foundCalendarID || !foundFrom || !foundTo {
						t.Errorf("events list should have --calendar-id, --from, and --to flags")
					}
				}
			}
		}
	}

	if !eventsListFound {
		t.Error("events list command not found")
	}
}

func TestRun_Behavior(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantExitCode    int
		wantStdoutParts []string
		wantStderr      string
	}{
		{
			name:            "no args shows help and exits non-zero",
			args:            []string{"caldav"},
			wantExitCode:    1,
			wantStdoutParts: []string{"NAME:", "caldav", "COMMANDS:", "calendars", "events"},
			wantStderr:      "",
		},
		{
			name:            "help flag keeps zero exit code",
			args:            []string{"caldav", "--help"},
			wantExitCode:    0,
			wantStdoutParts: []string{"NAME:", "caldav", "COMMANDS:"},
			wantStderr:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run(context.Background(), tt.args, &stdout, &stderr)

			if exitCode != tt.wantExitCode {
				t.Fatalf("run() exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			for _, part := range tt.wantStdoutParts {
				if !bytes.Contains(stdout.Bytes(), []byte(part)) {
					t.Fatalf("stdout = %q, want substring %q", stdout.String(), part)
				}
			}

			if stderr.String() != tt.wantStderr {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}
