package client

const RuntimeDir = "runtime"

func notnil[T any](t *T) *T {
	if t == nil {
		return new(T)
	}

	return t
}
