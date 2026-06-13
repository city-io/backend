package domain

import "time"

// NullTime is an optional timestamp. A nil Time represents the absence of a
// value.
type NullTime struct {
	*time.Time
}
