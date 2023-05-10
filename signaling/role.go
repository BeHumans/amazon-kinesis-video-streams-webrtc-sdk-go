package signaling

// Role Type of signaling connection actor
type Role string

// Signaling connection types
const (
	Master Role = "MASTER"
	Viewer Role = "VIEWER"
)

// DefaultClientID value when it not exist
const (
	DefaultClientID = "MASTER"
)
