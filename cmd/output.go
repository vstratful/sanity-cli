package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

// errQuiet is returned by emitError to signal a non-zero exit without
// re-printing the error to stderr (the JSON envelope already went to stdout).
var errQuiet = errors.New("sanity-cli: error already reported as JSON envelope")

// IsQuiet reports whether err is the sentinel returned by emitError.
func IsQuiet(err error) bool {
	return errors.Is(err, errQuiet)
}

// Envelope is the standard JSON output wrapper used by every command.
type Envelope struct {
	OK        bool        `json:"ok"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	CachedAt  string      `json:"cached_at,omitempty"`
	CachePath string      `json:"cache_path,omitempty"`
}

func writeJSON(w io.Writer, v interface{}, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func emitSuccess(data interface{}) error {
	return writeJSON(os.Stdout, Envelope{OK: true, Data: data}, pretty)
}

func emitSuccessRaw(data interface{}) error {
	return writeJSON(os.Stdout, data, pretty)
}

func emitSchemaSuccess(data interface{}, cachedAt, cachePath string) error {
	return writeJSON(os.Stdout, Envelope{
		OK:        true,
		Data:      data,
		CachedAt:  cachedAt,
		CachePath: cachePath,
	}, pretty)
}

// emitError writes an error envelope to stdout (JSON-first convention) and
// returns the quiet sentinel so the runner exits non-zero without re-printing.
func emitError(code, message string, details interface{}) error {
	_ = writeJSON(os.Stdout, Envelope{
		OK:      false,
		Error:   code,
		Message: message,
		Details: details,
	}, pretty)
	return errQuiet
}
