package status

type Counts struct {
	Source        string
	InstallID     string
	DriverVersion string
	AppVersion    string
	ConfigDigest  string
	ClaimAttempts int
	EmptyClaims   int
	ActiveLeaseID string
	WorkItemID    string
	ThreadID      string
	TurnID        string
	Acks          int
	Nacks         int
	Extensions    int
	LastErrorCode string
}
