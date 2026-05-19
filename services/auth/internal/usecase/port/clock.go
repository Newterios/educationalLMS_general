package port

import "time"

// Clock abstracts time.Now() so use-cases can be tested without real-time
// dependency. All use-case structs receive a Clock via their constructor;
// bare time.Now() calls inside use-cases are forbidden.
type Clock interface {
	Now() time.Time
}
