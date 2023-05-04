package signaling

// Type of signaling connection as
type Role string

// Signaling connection types
const (
	MASTER            Role = "MASTER"
	VIEWER            Role = "VIEWER"
	DEFAULT_CLIENT_ID Role = "MASTER"
)
