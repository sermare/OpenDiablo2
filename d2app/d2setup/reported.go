package d2setup

import "errors"

// ErrReported marks an error the user has already been told about in a dialog.
var ErrReported = errors.New("reported to the user")
