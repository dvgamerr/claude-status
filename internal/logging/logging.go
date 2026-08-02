// Package logging gives every claude-status subcommand a consistent, leveled
// zerolog.Logger instead of ad-hoc fmt.Fprintf(stderr, ...) calls, while
// keeping output human-readable (not raw JSON) since it's read directly on a
// terminal or tailed from a plain-text log file.
package logging

import (
	"io"

	"github.com/rs/zerolog"
)

// New returns a logger scoped to one subcommand (e.g. "ingest", "relay"),
// writing timestamped, leveled lines to w. NoColor is forced because w is
// often a redirected file (--log-file) or a piped stderr, where ANSI escapes
// would just be noise.
func New(w io.Writer, cmd string) zerolog.Logger {
	console := zerolog.ConsoleWriter{
		Out:        w,
		NoColor:    true,
		TimeFormat: "2006/01/02 15:04:05.000000",
	}
	return zerolog.New(console).With().Timestamp().Str("cmd", cmd).Logger()
}
