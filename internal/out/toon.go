package out

import (
	"context"
	"io"

	"github.com/toon-format/toon-go"
)

// toonRenderer outputs data in TOON format.
type toonRenderer struct{}

// Render outputs data in TOON format to the writer.
func (r *toonRenderer) Render(ctx context.Context, w io.Writer, data any) error {
	// Use toon.Marshal for simple cases
	bs, err := toon.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(bs)
	if err != nil {
		return err
	}
	// TOON doesn't add trailing newline, add one for consistency
	_, err = w.Write([]byte("\n"))
	return err
}

// RenderTOON renders data in TOON format to the writer.
func RenderTOON(ctx context.Context, w io.Writer, data any) error {
	return (&toonRenderer{}).Render(ctx, w, data)
}
