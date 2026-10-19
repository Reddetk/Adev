package repository

import "errors"

// ErrNotFound reports a missing record.
var ErrNotFound = errors.New("record not found")

// shortcut: the generated repository interfaces carry no context parameter, so the
// implementations use context.Background(). Upgrade when the use case layer passes
// its request context through.
