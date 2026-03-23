package out

import (
	"context"
	"io"
	"os"

	"golang.org/x/term"
)

// Format represents the output format available in the CLI.
type Format string

const (
	FormatJSON Format = "json"
	FormatTOON Format = "toon"
	FormatText Format = "text"
)

// DefaultFormatFromTTY determines the default output format based on whether stdout is a TTY.
// Returns "text" for TTY (interactive terminal), "toon" for non-TTY (pipes, CI, etc.).
func DefaultFormatFromTTY() Format {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return FormatText
	}
	return FormatTOON
}

// Renderer defines the interface for output formatters.
type Renderer interface {
	Render(ctx context.Context, w io.Writer, data any) error
}

// NewRenderer creates the appropriate renderer based on format.
func NewRenderer(format Format) Renderer {
	switch format {
	case FormatJSON:
		return &jsonRenderer{}
	case FormatTOON:
		return &toonRenderer{}
	case FormatText:
		return &textRenderer{}
	default:
		return &jsonRenderer{}
	}
}

// FormatResolver holds the format configuration for output.
type FormatResolver struct {
	ExplicitFormat Format
	IsTTY          bool
}

// NewFormatResolver creates a new FormatResolver with the given explicit format.
// If explicit format is empty, TTY detection will be used to determine the default.
func NewFormatResolver(explicitFormat string) *FormatResolver {
	f := ParseFormat(explicitFormat)
	if f != "" {
		return &FormatResolver{
			ExplicitFormat: f,
			IsTTY:          term.IsTerminal(int(os.Stdout.Fd())),
		}
	}
	return &FormatResolver{
		ExplicitFormat: "",
		IsTTY:          term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// Resolve returns the final format to use based on explicit setting or TTY detection.
func (r *FormatResolver) Resolve() Format {
	if r.ExplicitFormat != "" {
		return r.ExplicitFormat
	}
	if r.IsTTY {
		return FormatText
	}
	return FormatTOON
}

// ParseFormat parses a format string into a Format enum.
// Returns empty string for invalid/empty input (use default).
func ParseFormat(s string) Format {
	switch s {
	case "json", "JSON", "Json":
		return FormatJSON
	case "toon", "TOON", "Toon":
		return FormatTOON
	case "text", "TEXT", "Text":
		return FormatText
	default:
		return ""
	}
}

// Write outputs data using the specified format to stdout.
func Write(ctx context.Context, format Format, data any) error {
	return NewRenderer(format).Render(ctx, os.Stdout, data)
}

// WriteError outputs an error using the specified format to stderr.
func WriteError(ctx context.Context, format Format, err error) error {
	return NewRenderer(format).Render(ctx, os.Stderr, ErrorEnvelope{Error: err.Error()})
}

// ErrorEnvelope wraps error responses in a structured format.
type ErrorEnvelope struct {
	Error string `json:"error" toon:"error"`
}
