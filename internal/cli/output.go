package cli

import (
	"fmt"
	"io"

	"github.com/goccy/go-json"

	"github.com/asphaltbuffet/go-outside/internal/config"
)

// OutputWriter handles formatted output respecting JSON and quiet mode.
type OutputWriter struct {
	out       io.Writer
	err       io.Writer
	jsonMode  bool
	quietMode bool
}

// NewOutputWriterFromConfig creates an OutputWriter from a Config.
func NewOutputWriterFromConfig(out, err io.Writer, cfg *config.Config) *OutputWriter {
	return NewOutputWriter(out, err, cfg.IsJSON(), cfg.IsQuiet())
}

// NewOutputWriter creates an OutputWriter with explicit mode flags.
func NewOutputWriter(out, err io.Writer, jsonMode, quietMode bool) *OutputWriter {
	return &OutputWriter{
		out:       out,
		err:       err,
		jsonMode:  jsonMode,
		quietMode: quietMode,
	}
}

// IsJSON reports whether JSON mode is active.
func (w *OutputWriter) IsJSON() bool { return w.jsonMode }

// IsQuiet reports whether quiet mode is active.
func (w *OutputWriter) IsQuiet() bool { return w.quietMode }

// Success prints a success message to stdout (suppressed in quiet mode).
func (w *OutputWriter) Success(msg string) {
	if w.quietMode {
		return
	}
	fmt.Fprintln(w.out, msg)
}

// Error prints an error message to stderr.
func (w *OutputWriter) Error(msg string) {
	fmt.Fprintln(w.err, msg)
}

// JSON encodes v as JSON to stdout.
func (w *OutputWriter) JSON(v any) error {
	enc := json.NewEncoder(w.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
