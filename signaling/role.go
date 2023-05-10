package signaling

// Type of signaling connection as
type Role string

// Signaling connection types
const (
	Master          Role = "MASTER"
	Viewer          Role = "VIEWER"
	DefaultClientID Role = "MASTER"
)
