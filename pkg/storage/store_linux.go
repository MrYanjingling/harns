package storage

const (
	storePath = "/var/lib/edgeiot"
)

func isEphemeralError(err error) bool {
	return false
}
