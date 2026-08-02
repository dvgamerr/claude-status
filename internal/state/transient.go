package state

import "errors"

// ErrTransientRead marks a read failure as a short-lived race (a reader
// landing mid atomic-rename) rather than real corruption. See
// isTransientReadErr for the platform-specific detection and the comment on
// readRetryAttempts for why this happens at all.
var ErrTransientRead = errors.New("transient snapshot read conflict")
