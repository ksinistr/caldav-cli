package out

import (
	"context"
	"encoding/json"
	"io"
)

// jsonRenderer outputs data as JSON.
type jsonRenderer struct{}

// Render outputs data as formatted JSON to the writer.
func (r *jsonRenderer) Render(ctx context.Context, w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderJSON renders data as formatted JSON to the writer.
func RenderJSON(ctx context.Context, w io.Writer, data any) error {
	return (&jsonRenderer{}).Render(ctx, w, data)
}
