package client

const RuntimeDir = "runtime"

type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateExited  State = "exited"
)

// Returns the given value of type *T or,
// if it is nil, returns a new(T).
func newOr[T any](t *T) *T {
	if t == nil {
		return new(T)
	}

	return t
}
